package scanner

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dgit/internal/scanner/illustrator"
	"dgit/internal/scanner/photoshop"
)

// DesignFile contains metadata for detected design files
type DesignFile struct {
	Path       string   `json:"path"`        // Relative file path
	FileName   string   `json:"file_name"`   // Base filename
	Type       string   `json:"type"`        // File type: ai, psd, sketch, etc.
	Dimensions string   `json:"dimensions"`  // Canvas size: "1920x1080"
	ColorMode  string   `json:"color_mode"`  // Color space: RGB, CMYK, Grayscale
	Version    string   `json:"version"`     // Application version: "CC 2025 (29.x)"
	Layers     int      `json:"layers"`      // Number of layers in document
	Artboards  int      `json:"artboards"`   // Number of artboards/pages
	Objects    int      `json:"objects"`     // Estimated object count
	LayerNames []string `json:"layer_names"` // Names of all layers
	FileSize   int64    `json:"file_size"`   // File size in bytes

	// Cache Integration
	Hash       string        `json:"hash"`               // File hash for cache key generation
	CacheLevel string        `json:"cache_level"`        // Cache tier: hot/warm/cold
	Metadata   *FileMetadata `json:"metadata,omitempty"` // Pre-extracted metadata
	ScanTime   time.Duration `json:"scan_time"`          // Time taken to scan file
}

// FileMetadata contains pre-extracted design file metadata
type FileMetadata struct {
	Dimensions  string    `json:"dimensions,omitempty"`   // Canvas dimensions: "1920x1080"
	ColorMode   string    `json:"color_mode,omitempty"`   // Color space: RGB, CMYK
	Resolution  int       `json:"resolution,omitempty"`   // Document DPI
	LayerCount  int       `json:"layer_count,omitempty"`  // Number of layers
	FileVersion string    `json:"file_version,omitempty"` // Application version info
	ExtractedAt time.Time `json:"extracted_at"`           // Metadata extraction timestamp
}

// ScanResult contains scanning results with performance metrics
type ScanResult struct {
	TotalFiles    int
	DesignFiles   []DesignFile
	TypeCounts    map[string]int
	TotalSize     int64
	ErrorFiles    map[string]error
	ScanTime      time.Duration   `json:"scan_time"`      // Total scanning time
	CacheStats    *ScanCacheStats `json:"cache_stats"`    // Cache performance metrics
	MetadataStats *MetadataStats  `json:"metadata_stats"` // Metadata extraction statistics
}

// ScanCacheStats tracks scanning performance
type ScanCacheStats struct {
	FastScans      int `json:"fast_scans"`      // Files scanned in < 100ms
	SlowScans      int `json:"slow_scans"`      // Files taking longer to scan
	CacheableFiles int `json:"cacheable_files"` // Files suitable for hot cache
	MetadataHits   int `json:"metadata_hits"`   // Successful metadata extractions
}

// MetadataStats tracks metadata extraction performance by file type
type MetadataStats struct {
	PSDMetadata    int `json:"psd_metadata"`    // PSD files with extracted metadata
	AIMetadata     int `json:"ai_metadata"`     // AI files with extracted metadata
	SketchMetadata int `json:"sketch_metadata"` // Sketch files with extracted metadata
	OtherMetadata  int `json:"other_metadata"`  // Other files with extracted metadata
	FailedExtracts int `json:"failed_extracts"` // Failed metadata extractions
}

// FileScanner provides design file scanning and analysis capabilities
type FileScanner struct {
	supportedExts map[string]bool
	// Optimization Settings
	enableFastScan    bool  // Enable fast scanning mode for large files
	metadataThreshold int64 // File size threshold for metadata extraction (bytes)
}

// NewFileScanner creates a new standard FileScanner
func NewFileScanner() *FileScanner {
	return &FileScanner{
		supportedExts: map[string]bool{
			".ai":       true, // Adobe Illustrator
			".psd":      true, // Adobe Photoshop
			".sketch":   true, // Sketch App
			".fig":      true, // Figma (local files)
			".xd":       true, // Adobe XD
			".afdesign": true, // Affinity Designer
			".afphoto":  true, // Affinity Photo
			".blend":    true, // Blender
			".c4d":      true, // Cinema 4D
			".max":      true, // 3ds Max
			".mb":       true, // Maya Binary
			".ma":       true, // Maya ASCII
			".fbx":      true, // FBX
			".obj":      true, // OBJ
		},
		enableFastScan:    true,
		metadataThreshold: 500 * 1024 * 1024, // 500MB threshold for full analysis
	}
}

func NewFastFileScanner() *FileScanner {
	scanner := NewFileScanner()
	scanner.enableFastScan = true
	scanner.metadataThreshold = 100 * 1024 * 1024 // 100MB threshold
	return scanner
}

func (fs *FileScanner) ScanDirectory(folderPath string) (*ScanResult, error) {
	startTime := time.Now()

	result := &ScanResult{
		DesignFiles:   []DesignFile{},
		TypeCounts:    make(map[string]int),
		ErrorFiles:    make(map[string]error),
		CacheStats:    &ScanCacheStats{},
		MetadataStats: &MetadataStats{},
	}

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.ErrorFiles[path] = err
			return nil // Continue scanning despite errors
		}

		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == ".dgit" {
				return filepath.SkipDir
			}
			return nil
		}

		if IsDesignFile(path) {
			result.TotalFiles++
			result.TotalSize += info.Size()

			fileType := strings.ToLower(filepath.Ext(path)[1:])
			result.TypeCounts[fileType]++

			designFile, scanErr := fs.ScanFileWithPerformanceTracking(path, info)
			if scanErr != nil {
				result.ErrorFiles[path] = scanErr
				result.MetadataStats.FailedExtracts++
				// Create basic file info even if detailed scanning fails
				designFile = &DesignFile{
					Path:     path,
					FileName: info.Name(),
					Type:     fileType,
					FileSize: info.Size(),
					Hash:     fs.generateQuickHash(path, info),
				}
			}

			fs.updateScanStats(designFile, result.CacheStats, result.MetadataStats)

			result.DesignFiles = append(result.DesignFiles, *designFile)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	result.ScanTime = time.Since(startTime)
	return result, nil
}

// ScanFileWithPerformanceTracking scans individual files with performance metrics
func (fs *FileScanner) ScanFileWithPerformanceTracking(filePath string, info os.FileInfo) (*DesignFile, error) {
	startTime := time.Now()

	designFile, err := fs.ScanFile(filePath)
	if err != nil {
		return nil, err
	}

	designFile.ScanTime = time.Since(startTime)
	designFile.CacheLevel = fs.determineCacheLevel(info.Size(), designFile.ScanTime)

	return designFile, nil
}

// ScanFile performs analysis of individual design files
func (fs *FileScanner) ScanFile(filePath string) (*DesignFile, error) {
	if !IsDesignFile(filePath) {
		return nil, fmt.Errorf("unsupported file type")
	}

	fileName := filepath.Base(filePath)
	fileType := strings.ToLower(filepath.Ext(filePath)[1:])

	designFile := &DesignFile{
		Path:       filePath,
		FileName:   fileName,
		Type:       fileType,
		Dimensions: "Unknown",
		ColorMode:  "Unknown",
		Version:    "Unknown",
		Layers:     0,
		Artboards:  1,
		Objects:    0,
		LayerNames: []string{},
	}

	if info, err := os.Stat(filePath); err == nil {
		designFile.FileSize = info.Size()
		designFile.Hash = fs.generateFileHash(filePath, info)
	}

	// Skip heavy metadata extraction for large files
	if fs.enableFastScan && designFile.FileSize > fs.metadataThreshold {
		designFile.Metadata = &FileMetadata{
			ExtractedAt: time.Now(),
			FileVersion: fmt.Sprintf("Large %s file", strings.ToUpper(fileType)),
			Dimensions:  "Skipped (large file)",
			ColorMode:   "Unknown",
		}
		return designFile, nil
	}

	// Perform detailed analysis based on file type
	switch fileType {
	case "ai":
		return fs.analyzeAIFile(filePath, designFile)
	case "psd":
		return fs.analyzePSDFile(filePath, designFile)
	case "sketch":
		return fs.analyzeSketchFile(filePath, designFile)
	case "fig":
		return fs.analyzeFigmaFile(filePath, designFile)
	case "xd":
		return fs.analyzeXDFile(filePath, designFile)
	default:
		return designFile, nil
	}
}

// analyzeAIFile performs Adobe Illustrator file analysis
func (fs *FileScanner) analyzeAIFile(filePath string, designFile *DesignFile) (*DesignFile, error) {
	aiInfo, err := illustrator.GetAIInfo(filePath)
	if err != nil {
		return designFile, err // Return basic info even if detailed analysis fails
	}

	designFile.Dimensions = fmt.Sprintf("%dx%d px", aiInfo.Width, aiInfo.Height)
	designFile.ColorMode = aiInfo.ColorMode
	designFile.Version = aiInfo.Version
	designFile.Layers = aiInfo.LayerCount
	designFile.Artboards = aiInfo.ArtboardCount
	designFile.Objects = aiInfo.ObjectCount
	designFile.LayerNames = aiInfo.LayerNames

	designFile.Metadata = &FileMetadata{
		Dimensions:  designFile.Dimensions,
		ColorMode:   designFile.ColorMode,
		Resolution:  72, // Standard AI resolution
		LayerCount:  aiInfo.LayerCount,
		FileVersion: aiInfo.Version,
		ExtractedAt: time.Now(),
	}

	return designFile, nil
}

// analyzePSDFile performs Adobe Photoshop file analysis
func (fs *FileScanner) analyzePSDFile(filePath string, designFile *DesignFile) (*DesignFile, error) {
	psdInfo, err := photoshop.GetPSDInfo(filePath)
	if err != nil {
		return designFile, err
	}

	designFile.Dimensions = fmt.Sprintf("%dx%d px", psdInfo.Width, psdInfo.Height)
	designFile.ColorMode = fs.mapPSDColorMode(psdInfo.Channels, psdInfo.Bits)
	designFile.Version = "CC 2025" // PSD version extraction is complex, use default
	designFile.Layers = psdInfo.LayerCount
	designFile.LayerNames = psdInfo.LayerNames
	designFile.Objects = len(psdInfo.LayerNames) * 2 // Estimated object count

	designFile.Metadata = &FileMetadata{
		Dimensions:  designFile.Dimensions,
		ColorMode:   designFile.ColorMode,
		Resolution:  72, // Default resolution assumption
		LayerCount:  psdInfo.LayerCount,
		FileVersion: designFile.Version,
		ExtractedAt: time.Now(),
	}

	return designFile, nil
}

// analyzeSketchFile performs Sketch file analysis
func (fs *FileScanner) analyzeSketchFile(filePath string, designFile *DesignFile) (*DesignFile, error) {
	// Sketch files are ZIP archives requiring complex parsing
	designFile.Dimensions = "Unknown"
	designFile.ColorMode = "RGB"
	designFile.Version = "Sketch App"
	designFile.Layers = 1
	designFile.LayerNames = []string{"Sketch Layer"}

	designFile.Metadata = &FileMetadata{
		Dimensions:  "Unknown",
		ColorMode:   "RGB",
		Resolution:  72,
		LayerCount:  1,
		FileVersion: "Sketch App",
		ExtractedAt: time.Now(),
	}

	return designFile, nil
}

// analyzeFigmaFile performs Figma file analysis
func (fs *FileScanner) analyzeFigmaFile(filePath string, designFile *DesignFile) (*DesignFile, error) {
	designFile.Dimensions = "Unknown"
	designFile.ColorMode = "RGB"
	designFile.Version = "Figma"
	designFile.Layers = 1
	designFile.LayerNames = []string{"Figma Frame"}

	designFile.Metadata = &FileMetadata{
		Dimensions:  "Unknown",
		ColorMode:   "RGB",
		Resolution:  72,
		LayerCount:  1,
		FileVersion: "Figma",
		ExtractedAt: time.Now(),
	}

	return designFile, nil
}

// analyzeXDFile performs Adobe XD file analysis
func (fs *FileScanner) analyzeXDFile(filePath string, designFile *DesignFile) (*DesignFile, error) {
	designFile.Dimensions = "Unknown"
	designFile.ColorMode = "RGB"
	designFile.Version = "Adobe XD"
	designFile.Layers = 1
	designFile.LayerNames = []string{"XD Artboard"}

	designFile.Metadata = &FileMetadata{
		Dimensions:  "Unknown",
		ColorMode:   "RGB",
		Resolution:  72,
		LayerCount:  1,
		FileVersion: "Adobe XD",
		ExtractedAt: time.Now(),
	}

	return designFile, nil
}

// generateFileHash creates hash for file identification
func (fs *FileScanner) generateFileHash(filePath string, info os.FileInfo) string {
	hashInput := fmt.Sprintf("%s:%d:%d", filePath, info.Size(), info.ModTime().Unix())
	hash := sha256.Sum256([]byte(hashInput))
	return fmt.Sprintf("%x", hash)[:16] // First 16 characters
}

// generateQuickHash creates fast hash for error cases
func (fs *FileScanner) generateQuickHash(filePath string, info os.FileInfo) string {
	hashInput := fmt.Sprintf("%s:%d", filepath.Base(filePath), info.Size())
	hash := sha256.Sum256([]byte(hashInput))
	return fmt.Sprintf("%x", hash)[:8] // First 8 characters
}

// determineCacheLevel determines cache tier based on file characteristics
func (fs *FileScanner) determineCacheLevel(fileSize int64, scanTime time.Duration) string {
	if fileSize < 50*1024*1024 && scanTime < 100*time.Millisecond {
		return "hot"
	}

	if fileSize < 200*1024*1024 && scanTime < 500*time.Millisecond {
		return "warm"
	}

	return "cold"
}

// updateScanStats updates performance statistics
func (fs *FileScanner) updateScanStats(designFile *DesignFile, cacheStats *ScanCacheStats, metadataStats *MetadataStats) {
	if designFile.ScanTime < 100*time.Millisecond {
		cacheStats.FastScans++
	} else {
		cacheStats.SlowScans++
	}

	if designFile.CacheLevel == "hot" {
		cacheStats.CacheableFiles++
	}

	if designFile.Metadata != nil {
		cacheStats.MetadataHits++

		switch designFile.Type {
		case "psd":
			metadataStats.PSDMetadata++
		case "ai":
			metadataStats.AIMetadata++
		case "sketch":
			metadataStats.SketchMetadata++
		default:
			metadataStats.OtherMetadata++
		}
	}
}

// mapPSDColorMode maps PSD channel information to readable color mode names
func (fs *FileScanner) mapPSDColorMode(channels, bits int) string {
	switch channels {
	case 1:
		return "Grayscale"
	case 3:
		return "RGB"
	case 4:
		return "CMYK"
	default:
		return fmt.Sprintf("RGB (%d channels)", channels)
	}
}

// IsDesignFile checks if a file is a supported design file format
func IsDesignFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	supportedExts := map[string]bool{
		".ai":       true, // Adobe Illustrator
		".psd":      true, // Adobe Photoshop
		".sketch":   true, // Sketch App
		".fig":      true, // Figma
		".xd":       true, // Adobe XD
		".afdesign": true, // Affinity Designer
		".afphoto":  true, // Affinity Photo
		".blend":    true, // Blender
		".c4d":      true, // Cinema 4D
		".max":      true, // 3ds Max
		".mb":       true, // Maya Binary
		".ma":       true, // Maya ASCII
		".fbx":      true, // FBX
		".obj":      true, // OBJ
	}
	return supportedExts[ext]
}

// GetScanPerformanceReport generates performance analysis from scan results
func (fs *FileScanner) GetScanPerformanceReport(result *ScanResult) *ScanPerformanceReport {
	if result == nil || result.CacheStats == nil || result.MetadataStats == nil {
		return nil
	}

	report := &ScanPerformanceReport{
		TotalFiles:        result.TotalFiles,
		TotalScanTime:     result.ScanTime,
		AverageScanTime:   time.Duration(0),
		FastScanRatio:     0,
		MetadataHitRatio:  0,
		CacheDistribution: make(map[string]int),
		Recommendations:   []string{},
	}

	if result.TotalFiles > 0 {
		report.AverageScanTime = result.ScanTime / time.Duration(result.TotalFiles)

		totalScans := result.CacheStats.FastScans + result.CacheStats.SlowScans
		if totalScans > 0 {
			report.FastScanRatio = float64(result.CacheStats.FastScans) / float64(totalScans)
		}

		if result.TotalFiles > 0 {
			report.MetadataHitRatio = float64(result.CacheStats.MetadataHits) / float64(result.TotalFiles)
		}
	}

	for _, file := range result.DesignFiles {
		if file.CacheLevel != "" {
			report.CacheDistribution[file.CacheLevel]++
		}
	}

	// Generate optimization recommendations
	if report.FastScanRatio < 0.8 {
		report.Recommendations = append(report.Recommendations,
			"Consider enabling fast scan mode for large files")
	}
	if report.MetadataHitRatio < 0.7 {
		report.Recommendations = append(report.Recommendations,
			"Some files failed metadata extraction - check file integrity")
	}
	if report.CacheDistribution["hot"] < result.TotalFiles/2 {
		report.Recommendations = append(report.Recommendations,
			"Many files are not suitable for hot cache - consider file size optimization")
	}

	return report
}

// ScanPerformanceReport contains performance analysis of scan operations
type ScanPerformanceReport struct {
	TotalFiles        int            `json:"total_files"`        // Total number of files scanned
	TotalScanTime     time.Duration  `json:"total_scan_time"`    // Total time spent scanning
	AverageScanTime   time.Duration  `json:"average_scan_time"`  // Average time per file
	FastScanRatio     float64        `json:"fast_scan_ratio"`    // Ratio of fast scans (< 100ms)
	MetadataHitRatio  float64        `json:"metadata_hit_ratio"` // Successful metadata extractions
	CacheDistribution map[string]int `json:"cache_distribution"` // Files per cache tier
	Recommendations   []string       `json:"recommendations"`    // Optimization suggestions
}

// ============================================================================
// LEGACY COMPATIBILITY FUNCTIONS
// ============================================================================

// ScanFolder provides legacy API compatibility for existing code
func ScanFolder(folderPath string) ([]DesignFile, error) {
	scanner := NewFileScanner()
	result, err := scanner.ScanDirectory(folderPath)
	if err != nil {
		return nil, err
	}
	return result.DesignFiles, nil
}

// ScanFolderFast provides fast scanning for performance-critical operations
func ScanFolderFast(folderPath string) ([]DesignFile, error) {
	scanner := NewFastFileScanner()
	result, err := scanner.ScanDirectory(folderPath)
	if err != nil {
		return nil, err
	}
	return result.DesignFiles, nil
}
