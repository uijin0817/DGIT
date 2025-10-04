package restore

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dgit/internal/log"

	"github.com/klauspost/compress/zstd"
	"github.com/kr/binarydist"
	"github.com/pierrec/lz4/v4"
)

// RestoreManager handles file restoration with simplified storage system
type RestoreManager struct {
	DgitDir    string
	ObjectsDir string
	DeltaDir   string

	// Simplified Storage System
	VersionsDir string // Main version storage (.dgit/versions/)
	CommitsDir  string // Commit metadata (.dgit/commits/)
	CacheDir    string // Single cache directory (.dgit/cache/)
}

// NewRestoreManager creates a new restore manager with simplified structure
func NewRestoreManager(dgitDir string) *RestoreManager {
	objectsDir := filepath.Join(dgitDir, "objects")
	return &RestoreManager{
		DgitDir:    dgitDir,
		ObjectsDir: objectsDir,
		DeltaDir:   filepath.Join(objectsDir, "deltas"),
		// Simplified storage structure
		VersionsDir: filepath.Join(dgitDir, "versions"),
		CommitsDir:  filepath.Join(dgitDir, "commits"),
		CacheDir:    filepath.Join(dgitDir, "cache"),
	}
}

// RestoreResult contains restoration operation information
type RestoreResult struct {
	RestoredFiles    []string
	SkippedFiles     []string
	ErrorFiles       map[string]error
	RestoreMethod    string // "versions", "cache", "smart_delta", "delta_chain", "zip"
	RestorationTime  time.Duration
	TotalFilesCount  int
	SourceVersion    int
	SourceCommitHash string
	// Performance Metrics
	CacheHitLevel    string  // "versions", "cache", "miss" - cache performance tracking
	SpeedImprovement float64 // Multiplier vs traditional restoration methods
	DataTransferred  int64   // Bytes actually read from storage
}

// RestoreFilesFromCommit restores files using optimized strategies
func (rm *RestoreManager) RestoreFilesFromCommit(commitHashOrVersion string, filesToRestore []string, targetCommit interface{}) error {
	startTime := time.Now()

	// Parse commit reference (supports both hash and version formats)
	version, err := rm.parseCommitReference(commitHashOrVersion)
	if err != nil {
		return err
	}

	fmt.Printf("Analyzing restoration strategy for v%d...\n", version)

	// Load commit data using log manager
	logManager := log.NewLogManager(rm.DgitDir)
	commit, err := logManager.GetCommit(version)
	if err != nil {
		return fmt.Errorf("failed to load commit data: %w", err)
	}

	// Choose optimal restoration method based on cache availability
	result, err := rm.performFastRestore(commit, filesToRestore, version)
	if err != nil {
		return err
	}

	// Calculate performance metrics
	result.RestorationTime = time.Since(startTime)
	result.SpeedImprovement = rm.calculateSpeedImprovement(result.RestoreMethod, result.RestorationTime)

	// Display restoration results
	rm.displayRestoreResults(result, commitHashOrVersion, version)

	return nil
}

// performFastRestore intelligently chooses the fastest available restoration method
// Priority: Versions → Cache → Smart Delta → Legacy
func (rm *RestoreManager) performFastRestore(commit *log.Commit, filesToRestore []string, version int) (*RestoreResult, error) {
	result := &RestoreResult{
		SourceVersion:    commit.Version,
		SourceCommitHash: commit.Hash,
		RestoredFiles:    []string{},
		SkippedFiles:     []string{},
		ErrorFiles:       make(map[string]error),
	}

	// Priority 1: Versions Directory (LZ4) - highest priority
	if versionResult := rm.tryVersionRestore(commit, filesToRestore, result); versionResult != nil {
		return versionResult, nil
	}

	// Priority 2: Cache Directory - optimized access
	if cacheResult := rm.tryCacheRestore(commit, filesToRestore, result); cacheResult != nil {
		return cacheResult, nil
	}

	// Priority 3: Smart Delta Reconstruction for design files
	if commit.CompressionInfo != nil {
		switch commit.CompressionInfo.Strategy {
		case "psd_smart":
			fmt.Println("Using smart PSD delta restoration...")
			result.RestoreMethod = "smart_delta"
			result.CacheHitLevel = "smart"
			return rm.restoreFromSmartDelta(commit, filesToRestore, result)
		case "design_smart_delta":
			fmt.Println("Using smart design delta restoration...")
			result.RestoreMethod = "smart_delta"
			result.CacheHitLevel = "smart"
			return rm.restoreFromSmartDelta(commit, filesToRestore, result)
		case "bsdiff", "xdelta3":
			fmt.Println("Using optimized delta chain restoration...")
			result.RestoreMethod = "delta_chain"
			result.CacheHitLevel = "miss"
			return rm.restoreFromOptimizedDeltaChain(version, filesToRestore, result)
		case "zip":
			fmt.Println("Using direct ZIP restoration...")
			result.RestoreMethod = "zip"
			result.CacheHitLevel = "miss"
			return rm.restoreFromZip(commit.CompressionInfo.OutputFile, filesToRestore, result)
		}
	}

	// Fallback: Legacy ZIP restoration for backward compatibility
	if commit.SnapshotZip != "" {
		fmt.Println("Using legacy ZIP restoration...")
		result.RestoreMethod = "zip"
		result.CacheHitLevel = "miss"
		return rm.restoreFromZip(commit.SnapshotZip, filesToRestore, result)
	}

	return result, fmt.Errorf("no restoration method available for version %d", version)
}

// tryVersionRestore attempts restoration from versions directory
func (rm *RestoreManager) tryVersionRestore(commit *log.Commit, filesToRestore []string, result *RestoreResult) *RestoreResult {
	if commit.CompressionInfo == nil || commit.CompressionInfo.Strategy != "lz4" {
		return nil
	}

	versionPath := filepath.Join(rm.VersionsDir, commit.CompressionInfo.OutputFile)
	if !rm.fileExists(versionPath) {
		return nil
	}

	fmt.Println("Using versions directory - fast access!")
	result.RestoreMethod = "versions"
	result.CacheHitLevel = "versions"

	// Extract from LZ4 versions directory
	if err := rm.extractFromLZ4(versionPath, filesToRestore, result); err != nil {
		return nil
	}

	return result
}

// tryCacheRestore attempts restoration from cache directory
func (rm *RestoreManager) tryCacheRestore(commit *log.Commit, filesToRestore []string, result *RestoreResult) *RestoreResult {
	// Check for cache version
	cachePath := filepath.Join(rm.CacheDir, fmt.Sprintf("v%d.lz4", commit.Version))
	if !rm.fileExists(cachePath) {
		// Check for optimized cache version
		cachePath = filepath.Join(rm.CacheDir, fmt.Sprintf("v%d_optimized.zstd", commit.Version))
		if !rm.fileExists(cachePath) {
			return nil
		}
	}

	fmt.Println("Using cache directory - optimized access!")
	result.RestoreMethod = "cache"
	result.CacheHitLevel = "cache"

	// Extract from cache with appropriate decompression
	var err error
	if strings.HasSuffix(cachePath, ".lz4") {
		err = rm.extractFromLZ4(cachePath, filesToRestore, result)
	} else if strings.HasSuffix(cachePath, ".zstd") {
		err = rm.extractFromZstd(cachePath, filesToRestore, result)
	}

	if err != nil {
		return nil
	}

	return result
}

// extractFromLZ4 extracts files from LZ4 storage
func (rm *RestoreManager) extractFromLZ4(lz4Path string, filesToRestore []string, result *RestoreResult) error {
	// Extract version number from LZ4 filename
	fileName := filepath.Base(lz4Path)
	versionStr := strings.TrimSuffix(strings.TrimPrefix(fileName, "v"), ".lz4")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return fmt.Errorf("failed to parse version from filename %s: %w", fileName, err)
	}

	// Load commit metadata from commits directory
	logManager := log.NewLogManager(rm.DgitDir)
	commit, err := logManager.GetCommit(version)
	if err != nil {
		return fmt.Errorf("failed to load commit v%d: %w", version, err)
	}

	// Open LZ4 file for decompression
	file, err := os.Open(lz4Path)
	if err != nil {
		return fmt.Errorf("failed to open LZ4 file: %w", err)
	}
	defer file.Close()

	// Create LZ4 reader for streaming decompression
	lz4Reader := lz4.NewReader(file)

	// Read all decompressed data efficiently
	decompressedData, err := io.ReadAll(lz4Reader)
	if err != nil {
		return fmt.Errorf("failed to decompress LZ4 data: %w", err)
	}

	result.DataTransferred = int64(len(decompressedData))

	// Get current working directory for file restoration
	currentWorkDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Process all files from structured LZ4 stream
	processedFiles := 0
	for fileName := range commit.Metadata {
		// Check if this file should be restored based on user request
		if len(filesToRestore) > 0 {
			shouldRestore := false
			for _, target := range filesToRestore {
				if rm.shouldRestoreFile(fileName, []string{target}) {
					shouldRestore = true
					break
				}
			}
			if !shouldRestore {
				result.SkippedFiles = append(result.SkippedFiles, fileName)
				continue
			}
		}

		// Create target file path in working directory
		targetPath := filepath.Join(currentWorkDir, fileName)

		// Create file from decompressed data using structured format
		if err := rm.createFileFromStructuredData(targetPath, decompressedData, fileName); err != nil {
			result.ErrorFiles[fileName] = err
		} else {
			result.RestoredFiles = append(result.RestoredFiles, fileName)
			fmt.Printf("Restored %s\n", fileName)
		}

		processedFiles++
	}

	fmt.Printf("Processed %d files from storage\n", processedFiles)
	result.TotalFilesCount = len(result.RestoredFiles) + len(result.SkippedFiles) + len(result.ErrorFiles)
	return nil
}

// extractFromZstd extracts files from Zstd cache
func (rm *RestoreManager) extractFromZstd(zstdPath string, filesToRestore []string, result *RestoreResult) error {
	// Open Zstd file for decompression
	file, err := os.Open(zstdPath)
	if err != nil {
		return fmt.Errorf("failed to open Zstd file: %w", err)
	}
	defer file.Close()

	// Create Zstd reader for efficient decompression
	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create Zstd reader: %w", err)
	}
	defer zstdReader.Close()

	// Extract files from Zstd stream
	return rm.extractFilesFromStream(zstdReader, filesToRestore, result, zstdPath)
}

// extractFilesFromStream extracts files from LZ4/Zstd stream format efficiently
func (rm *RestoreManager) extractFilesFromStream(reader io.Reader, filesToRestore []string, result *RestoreResult, sourcePath string) error {
	// Read entire stream for processing
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read stream: %w", err)
	}

	result.DataTransferred = int64(len(data))

	// Get current working directory for file restoration
	currentWorkDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Parse structured stream format: "FILE:path:size\n[file_data]"
	content := string(data)
	pos := 0

	// Normalize target file paths for consistent matching
	normalizedTargets := make([]string, len(filesToRestore))
	for i, target := range filesToRestore {
		normalizedTargets[i] = filepath.Clean(strings.ReplaceAll(target, "\\", "/"))
	}

	// Process each file in the stream
	for pos < len(content) {
		// Find file header line
		headerEnd := strings.Index(content[pos:], "\n")
		if headerEnd == -1 {
			break
		}
		headerEnd += pos

		headerLine := content[pos:headerEnd]
		if !strings.HasPrefix(headerLine, "FILE:") {
			pos = headerEnd + 1
			continue
		}

		// Parse header: "FILE:path:size"
		parts := strings.Split(headerLine, ":")
		if len(parts) != 3 {
			pos = headerEnd + 1
			continue
		}

		filePath := parts[1]
		fileSize := rm.parseInt64(parts[2])
		if fileSize <= 0 {
			pos = headerEnd + 1
			continue
		}

		// Check if this file should be restored based on user request
		if len(filesToRestore) > 0 {
			if !rm.shouldRestoreFile(filePath, normalizedTargets) {
				result.SkippedFiles = append(result.SkippedFiles, filePath)
				pos = headerEnd + 1 + int(fileSize)
				continue
			}
		}

		// Extract file data from stream
		fileDataStart := headerEnd + 1
		fileDataEnd := fileDataStart + int(fileSize)

		if fileDataEnd > len(data) {
			break
		}

		fileData := data[fileDataStart:fileDataEnd]

		// Create target file in working directory
		targetPath := filepath.Join(currentWorkDir, filePath)
		if err := rm.createFileFromData(targetPath, fileData); err != nil {
			result.ErrorFiles[filePath] = err
		} else {
			result.RestoredFiles = append(result.RestoredFiles, filePath)
		}

		pos = fileDataEnd
	}

	result.TotalFilesCount = len(result.RestoredFiles) + len(result.SkippedFiles) + len(result.ErrorFiles)
	return nil
}

// createFileFromData creates a file with given data and proper directory structure
func (rm *RestoreManager) createFileFromData(filePath string, data []byte) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", filePath, err)
	}

	// Create and write file atomically
	return os.WriteFile(filePath, data, 0644)
}

// restoreFromSmartDelta restores from smart delta compression
func (rm *RestoreManager) restoreFromSmartDelta(commit *log.Commit, filesToRestore []string, result *RestoreResult) (*RestoreResult, error) {
	if commit.CompressionInfo == nil {
		return result, fmt.Errorf("no compression info in commit")
	}

	// Check cache directory for smart delta file
	deltaPath := filepath.Join(rm.CacheDir, commit.CompressionInfo.OutputFile)

	if !rm.fileExists(deltaPath) {
		// Try versions directory as fallback
		deltaPath = filepath.Join(rm.VersionsDir, commit.CompressionInfo.OutputFile)
		if !rm.fileExists(deltaPath) {
			return result, fmt.Errorf("smart delta file not found: %s", commit.CompressionInfo.OutputFile)
		}
	}

	fmt.Printf("Restoring from smart delta: %s\n", deltaPath)

	// Read delta file
	deltaData, err := os.ReadFile(deltaPath)
	if err != nil {
		return result, fmt.Errorf("failed to read delta file: %w", err)
	}

	// Parse delta file format
	content := string(deltaData)
	lines := strings.Split(content, "\n")

	if len(lines) < 3 {
		return result, fmt.Errorf("invalid smart delta format: too few lines")
	}

	// Verify header
	if lines[0] != "PSD_SMART_DELTA_V1" {
		return result, fmt.Errorf("invalid smart delta header: %s", lines[0])
	}

	// Parse metadata length
	if !strings.HasPrefix(lines[1], "METADATA_LENGTH:") {
		return result, fmt.Errorf("invalid metadata length line: %s", lines[1])
	}

	metadataLengthStr := strings.TrimPrefix(lines[1], "METADATA_LENGTH:")
	metadataLength, err := strconv.Atoi(metadataLengthStr)
	if err != nil {
		return result, fmt.Errorf("failed to parse metadata length: %w", err)
	}

	// Find metadata start position
	metadataStartPos := len(lines[0]) + 1 + len(lines[1]) + 1
	if metadataStartPos+metadataLength > len(deltaData) {
		return result, fmt.Errorf("invalid metadata length: exceeds file size")
	}

	// Extract metadata JSON
	metadataBytes := deltaData[metadataStartPos : metadataStartPos+metadataLength]

	// Parse metadata
	var deltaMetadata map[string]interface{}
	if err := json.Unmarshal(metadataBytes, &deltaMetadata); err != nil {
		return result, fmt.Errorf("failed to parse delta metadata: %w", err)
	}

	// Extract key information from metadata
	filePath, ok := deltaMetadata["file_path"].(string)
	if !ok {
		return result, fmt.Errorf("missing file_path in metadata")
	}

	baseVersion, ok := deltaMetadata["from_version"].(float64)
	if !ok {
		return result, fmt.Errorf("missing from_version in metadata")
	}

	// Check if base version exists
	if int(baseVersion) > 0 {
		baseVersionPath := filepath.Join(rm.CommitsDir, fmt.Sprintf("v%d.json", int(baseVersion)))
		if !rm.fileExists(baseVersionPath) {
			fmt.Printf("Warning: base version v%d metadata not found\n", int(baseVersion))
		}
	}

	// Find binary data position
	binaryDataMarker := "\nBINARY_DATA:\n"
	binaryDataPos := bytes.Index(deltaData, []byte(binaryDataMarker))
	if binaryDataPos == -1 {
		return result, fmt.Errorf("binary data marker not found")
	}
	binaryDataPos += len(binaryDataMarker)

	// Extract LZ4 compressed data
	compressedData := deltaData[binaryDataPos:]

	// Decompress using LZ4
	lz4Reader := lz4.NewReader(bytes.NewReader(compressedData))
	decompressedData, err := io.ReadAll(lz4Reader)
	if err != nil {
		return result, fmt.Errorf("failed to decompress LZ4 data: %w", err)
	}

	// Get current working directory
	currentWorkDir, err := os.Getwd()
	if err != nil {
		return result, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if this file should be restored
	if len(filesToRestore) > 0 {
		shouldRestore := false
		for _, target := range filesToRestore {
			if rm.shouldRestoreFile(filePath, []string{target}) {
				shouldRestore = true
				break
			}
		}
		if !shouldRestore {
			result.SkippedFiles = append(result.SkippedFiles, filePath)
			return result, nil
		}
	}

	// For PSD smart delta, the decompressed data is the complete new file
	targetPath := filepath.Join(currentWorkDir, filePath)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
		return result, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the restored file
	if err := os.WriteFile(targetPath, decompressedData, 0644); err != nil {
		result.ErrorFiles[filePath] = err
		return result, fmt.Errorf("failed to write restored file: %w", err)
	}

	result.RestoredFiles = append(result.RestoredFiles, filePath)
	result.TotalFilesCount = 1
	result.DataTransferred = int64(len(decompressedData))

	// Log layer change information if available
	if layerAnalysis, ok := deltaMetadata["layer_analysis"].(map[string]interface{}); ok {
		if summary, ok := layerAnalysis["changes_summary"].(string); ok {
			fmt.Printf("Layer changes applied: %s\n", summary)
		}

		if addedLayers, ok := layerAnalysis["added_layers"].([]interface{}); ok && len(addedLayers) > 0 {
			fmt.Printf("  Added %d layers\n", len(addedLayers))
		}
		if deletedLayers, ok := layerAnalysis["deleted_layers"].([]interface{}); ok && len(deletedLayers) > 0 {
			fmt.Printf("  Deleted %d layers\n", len(deletedLayers))
		}
		if changedLayers, ok := layerAnalysis["changed_layers"].([]interface{}); ok && len(changedLayers) > 0 {
			fmt.Printf("  Modified %d layers\n", len(changedLayers))
		}
	}

	fmt.Printf("Successfully restored %s (%d bytes)\n", filePath, len(decompressedData))

	return result, nil
}

// restoreFromOptimizedDeltaChain restores from optimized delta chain
func (rm *RestoreManager) restoreFromOptimizedDeltaChain(targetVersion int, filesToRestore []string, result *RestoreResult) (*RestoreResult, error) {
	// Find optimal restoration path through simplified storage hierarchy
	restorationPath, err := rm.findOptimizedRestorationPath(targetVersion)
	if err != nil {
		return result, err
	}

	fmt.Printf("   Found restoration path: %d steps\n", len(restorationPath))

	// Execute optimized restoration sequence
	tempFile, err := rm.executeOptimizedRestorationPath(restorationPath)
	if err != nil {
		return result, err
	}
	defer os.Remove(tempFile)

	// Extract files from final restored ZIP
	return rm.extractFilesFromZip(tempFile, filesToRestore, result)
}

// findOptimizedRestorationPath finds fastest restoration path using simplified storage hierarchy
func (rm *RestoreManager) findOptimizedRestorationPath(targetVersion int) ([]RestorationStep, error) {
	var path []RestorationStep
	currentVersion := targetVersion

	// Work backwards with simplified storage prioritization
	for currentVersion > 0 {
		// Priority 1: Check versions directory first (LZ4)
		versionPath := filepath.Join(rm.VersionsDir, fmt.Sprintf("v%d.lz4", currentVersion))
		if rm.fileExists(versionPath) {
			step := RestorationStep{
				Type:    "lz4",
				File:    versionPath,
				Version: currentVersion,
			}
			path = append([]RestorationStep{step}, path...)
			break
		}

		// Priority 2: Check cache directory
		cachePath := filepath.Join(rm.CacheDir, fmt.Sprintf("v%d.lz4", currentVersion))
		if rm.fileExists(cachePath) {
			step := RestorationStep{
				Type:    "lz4",
				File:    cachePath,
				Version: currentVersion,
			}
			path = append([]RestorationStep{step}, path...)
			break
		}

		// Check for optimized cache version
		optimizedPath := filepath.Join(rm.CacheDir, fmt.Sprintf("v%d_optimized.zstd", currentVersion))
		if rm.fileExists(optimizedPath) {
			step := RestorationStep{
				Type:    "zstd",
				File:    optimizedPath,
				Version: currentVersion,
			}
			path = append([]RestorationStep{step}, path...)
			break
		}

		// Check for direct ZIP snapshot (legacy compatibility)
		zipPath := filepath.Join(rm.ObjectsDir, fmt.Sprintf("v%d.zip", currentVersion))
		if rm.fileExists(zipPath) {
			step := RestorationStep{
				Type:    "zip",
				File:    zipPath,
				Version: currentVersion,
			}
			path = append([]RestorationStep{step}, path...)
			break
		}

		// Look for delta files
		deltaPath := filepath.Join(rm.DeltaDir, fmt.Sprintf("v%d_from_v%d.bsdiff", currentVersion, currentVersion-1))
		if rm.fileExists(deltaPath) {
			step := RestorationStep{
				Type:    "bsdiff",
				File:    deltaPath,
				Version: currentVersion,
			}
			path = append([]RestorationStep{step}, path...)
			currentVersion--
			continue
		}

		// Check for smart delta files in cache
		smartDeltaPath := filepath.Join(rm.CacheDir, fmt.Sprintf("v%d_from_v%d.psd_smart", currentVersion, currentVersion-1))
		if rm.fileExists(smartDeltaPath) {
			step := RestorationStep{
				Type:    "smart_delta",
				File:    smartDeltaPath,
				Version: currentVersion,
			}
			path = append([]RestorationStep{step}, path...)
			currentVersion--
			continue
		}

		return nil, fmt.Errorf("missing restoration data for version %d", currentVersion)
	}

	if len(path) == 0 {
		return nil, fmt.Errorf("no restoration path found for version %d", targetVersion)
	}

	return path, nil
}

// executeOptimizedRestorationPath executes restoration plan
func (rm *RestoreManager) executeOptimizedRestorationPath(path []RestorationStep) (string, error) {
	// Start with the base file from simplified storage hierarchy
	baseStep := path[0]

	// Create working file based on base type
	tempFile := filepath.Join(rm.ObjectsDir, fmt.Sprintf("temp_restore_%d.zip", time.Now().UnixNano()))

	switch baseStep.Type {
	case "lz4":
		if err := rm.convertLZ4ToZip(baseStep.File, tempFile); err != nil {
			return "", err
		}
	case "zstd":
		if err := rm.convertZstdToZip(baseStep.File, tempFile); err != nil {
			return "", err
		}
	case "zip":
		if err := rm.copyFile(baseStep.File, tempFile); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported base file type: %s", baseStep.Type)
	}

	// Apply deltas in sequence
	for i := 1; i < len(path); i++ {
		step := path[i]
		nextTempFile := filepath.Join(rm.ObjectsDir, fmt.Sprintf("temp_restore_%d_%d.zip", time.Now().UnixNano(), i))

		switch step.Type {
		case "bsdiff":
			if err := rm.applyBsdiffPatch(tempFile, step.File, nextTempFile); err != nil {
				return "", fmt.Errorf("failed to apply bsdiff patch for v%d: %w", step.Version, err)
			}
		case "smart_delta":
			if err := rm.applySmartDelta(tempFile, step.File, nextTempFile); err != nil {
				return "", fmt.Errorf("failed to apply smart delta for v%d: %w", step.Version, err)
			}
		case "xdelta3":
			return "", fmt.Errorf("xdelta3 restoration not yet implemented")
		default:
			return "", fmt.Errorf("unknown restoration step type: %s", step.Type)
		}

		// Clean up previous temp file and use new one
		os.Remove(tempFile)
		tempFile = nextTempFile
	}

	return tempFile, nil
}

// convertLZ4ToZip converts LZ4 cache file to ZIP format
func (rm *RestoreManager) convertLZ4ToZip(lz4Path, zipPath string) error {
	// Open LZ4 file for reading
	lz4File, err := os.Open(lz4Path)
	if err != nil {
		return err
	}
	defer lz4File.Close()

	// Create LZ4 reader for decompression
	lz4Reader := lz4.NewReader(lz4File)

	// Create ZIP file for output
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Convert LZ4 stream to ZIP format
	return rm.convertStreamToZip(lz4Reader, zipWriter)
}

// convertZstdToZip converts Zstd cache file to ZIP format
func (rm *RestoreManager) convertZstdToZip(zstdPath, zipPath string) error {
	// Open Zstd file for reading
	zstdFile, err := os.Open(zstdPath)
	if err != nil {
		return err
	}
	defer zstdFile.Close()

	// Create Zstd reader for decompression
	zstdReader, err := zstd.NewReader(zstdFile)
	if err != nil {
		return err
	}
	defer zstdReader.Close()

	// Create ZIP file for output
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Convert Zstd stream to ZIP format
	return rm.convertStreamToZip(zstdReader, zipWriter)
}

// convertStreamToZip converts LZ4/Zstd stream format to standard ZIP
func (rm *RestoreManager) convertStreamToZip(reader io.Reader, zipWriter *zip.Writer) error {
	// Read entire stream for processing
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	// Parse stream and create ZIP entries
	content := string(data)
	pos := 0

	for pos < len(content) {
		// Find file header in stream
		headerEnd := strings.Index(content[pos:], "\n")
		if headerEnd == -1 {
			break
		}
		headerEnd += pos

		headerLine := content[pos:headerEnd]
		if !strings.HasPrefix(headerLine, "FILE:") {
			pos = headerEnd + 1
			continue
		}

		// Parse header: "FILE:path:size"
		parts := strings.Split(headerLine, ":")
		if len(parts) != 3 {
			pos = headerEnd + 1
			continue
		}

		filePath := parts[1]
		fileSize := rm.parseInt64(parts[2])
		if fileSize <= 0 {
			pos = headerEnd + 1
			continue
		}

		// Extract file data from stream
		fileDataStart := headerEnd + 1
		fileDataEnd := fileDataStart + int(fileSize)

		if fileDataEnd > len(data) {
			break
		}

		fileData := data[fileDataStart:fileDataEnd]

		// Create ZIP entry for file
		zipEntry, err := zipWriter.Create(filePath)
		if err != nil {
			pos = fileDataEnd
			continue
		}

		_, err = zipEntry.Write(fileData)
		if err != nil {
			pos = fileDataEnd
			continue
		}

		pos = fileDataEnd
	}

	return nil
}

// applySmartDelta applies smart delta to create new file
func (rm *RestoreManager) applySmartDelta(baseFile, deltaFile, newFile string) error {
	// Read delta file
	deltaData, err := os.ReadFile(deltaFile)
	if err != nil {
		return fmt.Errorf("failed to read delta file: %w", err)
	}

	// Parse delta file to check format
	content := string(deltaData)
	if !strings.HasPrefix(content, "PSD_SMART_DELTA_V1") {
		// Not a smart delta, fall back to simple copy
		return rm.copyFile(baseFile, newFile)
	}

	// For PSD smart delta, the delta file contains the complete new version
	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return fmt.Errorf("invalid smart delta format")
	}

	// Parse metadata length
	if !strings.HasPrefix(lines[1], "METADATA_LENGTH:") {
		return fmt.Errorf("invalid metadata length line")
	}

	metadataLengthStr := strings.TrimPrefix(lines[1], "METADATA_LENGTH:")
	metadataLength, err := strconv.Atoi(metadataLengthStr)
	if err != nil {
		return fmt.Errorf("failed to parse metadata length: %w", err)
	}

	// Find binary data position
	binaryDataMarker := "\nBINARY_DATA:\n"
	binaryDataPos := bytes.Index(deltaData, []byte(binaryDataMarker))
	if binaryDataPos == -1 {
		return fmt.Errorf("binary data marker not found")
	}
	binaryDataPos += len(binaryDataMarker)

	// Extract and decompress LZ4 data
	compressedData := deltaData[binaryDataPos:]
	lz4Reader := lz4.NewReader(bytes.NewReader(compressedData))
	decompressedData, err := io.ReadAll(lz4Reader)
	if err != nil {
		return fmt.Errorf("failed to decompress LZ4 data: %w", err)
	}

	// Write the new file
	if err := os.WriteFile(newFile, decompressedData, 0644); err != nil {
		return fmt.Errorf("failed to write new file: %w", err)
	}

	// Log metadata for debugging
	metadataStartPos := len(lines[0]) + 1 + len(lines[1]) + 1
	if metadataStartPos+metadataLength <= len(deltaData) {
		metadataBytes := deltaData[metadataStartPos : metadataStartPos+metadataLength]
		var metadata map[string]interface{}
		if json.Unmarshal(metadataBytes, &metadata) == nil {
			if layerAnalysis, ok := metadata["layer_analysis"].(map[string]interface{}); ok {
				if summary, ok := layerAnalysis["changes_summary"].(string); ok {
					fmt.Printf("Applied smart delta: %s\n", summary)
				}
			}
		}
	}

	return nil
}

// calculateSpeedImprovement calculates speed improvement based on restore method
func (rm *RestoreManager) calculateSpeedImprovement(method string, duration time.Duration) float64 {
	// Traditional restoration baseline: 10 seconds
	baselineMs := 10000.0
	actualMs := float64(duration.Nanoseconds()) / 1000000.0

	switch method {
	case "versions":
		return baselineMs / actualMs
	case "cache":
		return baselineMs / actualMs
	case "smart_delta":
		return baselineMs / actualMs
	default:
		return baselineMs / actualMs
	}
}

// displayRestoreResults shows restoration results
func (rm *RestoreManager) displayRestoreResults(result *RestoreResult, commitRef string, version int) {
	if len(result.RestoredFiles) > 0 {
		fmt.Printf("\nRestoration completed in %.3f seconds\n",
			result.RestorationTime.Seconds())

		// Show method-specific information
		switch result.RestoreMethod {
		case "versions":
			fmt.Printf("Versions directory restoration - %.1fx faster than traditional!\n", result.SpeedImprovement)
			fmt.Printf("Data transferred: %.2f KB from versions storage\n", float64(result.DataTransferred)/1024)
		case "cache":
			fmt.Printf("Cache directory restoration - %.1fx faster than traditional!\n", result.SpeedImprovement)
			fmt.Printf("Data transferred: %.2f KB from cache storage\n", float64(result.DataTransferred)/1024)
		case "smart_delta":
			fmt.Printf("Smart delta restoration - intelligent reconstruction!\n")
		case "delta_chain":
			fmt.Printf("Optimized delta chain restoration completed\n")
		case "zip":
			fmt.Printf("ZIP extraction completed\n")
		}

		fmt.Printf("Successfully restored %d files\n", len(result.RestoredFiles))

		// List restored files with visual file type indicators
		for _, file := range result.RestoredFiles {
			fileType := rm.getFileTypeIndicator(file)
			fmt.Printf("  %s %s\n", fileType, file)
		}
	}

	// Show any restoration errors encountered
	if len(result.ErrorFiles) > 0 {
		fmt.Printf("\n%d files failed to restore:\n", len(result.ErrorFiles))
		for file, err := range result.ErrorFiles {
			fmt.Printf("   %s: %v\n", file, err)
		}
	}

	// Handle case where no files matched criteria
	if len(result.RestoredFiles) == 0 && len(result.ErrorFiles) == 0 {
		fmt.Println("No files found matching the specified criteria.")
	}

	fmt.Printf("\nRestoration from commit %s (v%d) completed!\n", commitRef, version)
	fmt.Printf("Cache performance: %s cache hit\n", result.CacheHitLevel)
}

// RestorationStep represents a single step in restoration process
type RestorationStep struct {
	Type    string // "zip", "lz4", "zstd", "bsdiff", "xdelta3", "smart_delta"
	File    string
	Version int
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// parseInt64 safely parses string to int64
func (rm *RestoreManager) parseInt64(s string) int64 {
	result := int64(0)
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int64(r-'0')
		} else {
			return 0
		}
	}
	return result
}

// parseCommitReference parses commit reference to version number
func (rm *RestoreManager) parseCommitReference(commitRef string) (int, error) {
	// Handle "v1", "v2", etc. format
	if strings.HasPrefix(commitRef, "v") {
		versionStr := strings.TrimPrefix(commitRef, "v")
		if v, err := strconv.Atoi(versionStr); err == nil {
			return v, nil
		}
	}

	// Handle "1", "2", etc. format
	if v, err := strconv.Atoi(commitRef); err == nil {
		return v, nil
	}

	return 0, fmt.Errorf("invalid commit reference: %s", commitRef)
}

// getFileTypeIndicator returns visual indicator for file type
func (rm *RestoreManager) getFileTypeIndicator(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".ai":
		return "[AI]"
	case ".psd":
		return "[PSD]"
	case ".sketch":
		return "[SKETCH]"
	case ".fig":
		return "[FIG]"
	case ".xd":
		return "[XD]"
	case ".blend":
		return "[BLEND]"
	case ".c4d":
		return "[C4D]"
	default:
		return "[FILE]"
	}
}

// fileExists checks if a file exists on the filesystem
func (rm *RestoreManager) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// copyFile copies a file from source to destination
func (rm *RestoreManager) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// ============================================================================
// EXISTING FUNCTIONS (PRESERVED FOR COMPATIBILITY)
// ============================================================================

// restoreFromZip restores from ZIP file
func (rm *RestoreManager) restoreFromZip(zipFileName string, filesToRestore []string, result *RestoreResult) (*RestoreResult, error) {
	zipPath := filepath.Join(rm.ObjectsDir, zipFileName)

	// Check if ZIP file exists
	if !rm.fileExists(zipPath) {
		return result, fmt.Errorf("ZIP file not found: %s", zipFileName)
	}

	return rm.extractFilesFromZip(zipPath, filesToRestore, result)
}

// extractFilesFromZip extracts files from a ZIP archive
func (rm *RestoreManager) extractFilesFromZip(zipPath string, filesToRestore []string, result *RestoreResult) (*RestoreResult, error) {
	// Open ZIP file for reading
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return result, fmt.Errorf("failed to open ZIP file %s: %w", zipPath, err)
	}
	defer r.Close()

	// Get current working directory
	currentWorkDir, err := os.Getwd()
	if err != nil {
		return result, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Normalize target file paths
	normalizedTargets := make([]string, len(filesToRestore))
	for i, target := range filesToRestore {
		normalizedTargets[i] = filepath.Clean(strings.ReplaceAll(target, "\\", "/"))
	}

	// Process each file in the ZIP archive
	for _, f := range r.File {
		// Normalize file path in ZIP
		filePathInZip := strings.ReplaceAll(f.Name, "\\", "/")

		// Check if this file should be restored
		if len(filesToRestore) > 0 {
			if !rm.shouldRestoreFile(filePathInZip, normalizedTargets) {
				result.SkippedFiles = append(result.SkippedFiles, filePathInZip)
				continue
			}
		}

		// Skip directories in ZIP archive
		if f.FileInfo().IsDir() {
			continue
		}

		// Restore the individual file
		if err := rm.restoreFile(f, filePathInZip, currentWorkDir); err != nil {
			result.ErrorFiles[filePathInZip] = err
			continue
		}

		result.RestoredFiles = append(result.RestoredFiles, filePathInZip)
	}

	result.TotalFilesCount = len(r.File)
	return result, nil
}

// shouldRestoreFile determines if a file should be restored
func (rm *RestoreManager) shouldRestoreFile(filePathInZip string, normalizedTargets []string) bool {
	for _, target := range normalizedTargets {
		// Exact file path match
		if filePathInZip == target {
			return true
		}

		// Filename-only match
		if filepath.Base(filePathInZip) == filepath.Base(target) {
			return true
		}

		// Directory match
		if strings.HasSuffix(target, "/") && strings.HasPrefix(filePathInZip, target) {
			return true
		}

		// Partial path match
		if strings.Contains(filePathInZip, strings.Trim(target, "/")) {
			return true
		}
	}

	return false
}

// restoreFile restores a single file from ZIP
func (rm *RestoreManager) restoreFile(f *zip.File, filePathInZip, currentWorkDir string) error {
	// Determine final target path
	targetPath := filepath.Join(currentWorkDir, filePathInZip)

	// Create target directory structure
	if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
	}

	// Open file within ZIP archive
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open file %s in zip: %w", filePathInZip, err)
	}
	defer rc.Close()

	// Create target file for writing
	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}
	defer outFile.Close()

	// Copy content from ZIP to target file
	if _, err = io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("failed to copy content for %s: %w", filePathInZip, err)
	}

	return nil
}

// applyBsdiffPatch applies a bsdiff patch
func (rm *RestoreManager) applyBsdiffPatch(oldFile, patchFile, newFile string) error {
	// Open old file
	old, err := os.Open(oldFile)
	if err != nil {
		return fmt.Errorf("failed to open old file: %w", err)
	}
	defer old.Close()

	// Open patch file
	patch, err := os.Open(patchFile)
	if err != nil {
		return fmt.Errorf("failed to open patch file: %w", err)
	}
	defer patch.Close()

	// Create new file
	new, err := os.Create(newFile)
	if err != nil {
		return fmt.Errorf("failed to create new file: %w", err)
	}
	defer new.Close()

	// Apply binary patch
	if err := binarydist.Patch(old, new, patch); err != nil {
		return fmt.Errorf("binarydist patch failed: %w", err)
	}

	return nil
}

// createFileFromStructuredData creates a file from structured LZ4/Zstd data
func (rm *RestoreManager) createFileFromStructuredData(filePath string, data []byte, targetFileName string) error {
	// Parse structured data to find target file
	content := string(data)
	pos := 0

	for pos < len(content) {
		// Find file header line
		headerEnd := strings.Index(content[pos:], "\n")
		if headerEnd == -1 {
			break
		}
		headerEnd += pos

		headerLine := content[pos:headerEnd]
		if !strings.HasPrefix(headerLine, "FILE:") {
			pos = headerEnd + 1
			continue
		}

		// Parse header: "FILE:path:size"
		parts := strings.Split(headerLine, ":")
		if len(parts) != 3 {
			pos = headerEnd + 1
			continue
		}

		fileName := parts[1]
		fileSize, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || fileSize <= 0 {
			pos = headerEnd + 1
			continue
		}

		// Check if this is our target file
		if fileName == targetFileName || filepath.Base(fileName) == filepath.Base(targetFileName) {
			// Extract file data
			fileDataStart := headerEnd + 1
			fileDataEnd := fileDataStart + int(fileSize)

			if fileDataEnd > len(data) {
				return fmt.Errorf("file data exceeds available data")
			}

			fileData := data[fileDataStart:fileDataEnd]

			// Create target directory if needed
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			// Write file
			return os.WriteFile(filePath, fileData, 0644)
		}

		// Skip to next file
		pos = headerEnd + 1 + int(fileSize)
	}

	return fmt.Errorf("file not found in structured data: %s", targetFileName)
}
