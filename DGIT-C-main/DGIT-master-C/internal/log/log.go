package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CompressionResult contains comprehensive compression operation results
// Enhanced with performance metrics
type CompressionResult struct {
	Strategy         string    `json:"strategy"` // "lz4", "zip", "bsdiff", "xdelta3", "psd_smart"
	OutputFile       string    `json:"output_file"`
	OriginalSize     int64     `json:"original_size"`
	CompressedSize   int64     `json:"compressed_size"`
	CompressionRatio float64   `json:"compression_ratio"`
	BaseVersion      int       `json:"base_version,omitempty"`
	CreatedAt        time.Time `json:"created_at"`

	// Performance Metrics - Core data for speed improvement tracking
	CompressionTime  float64 `json:"compression_time_ms"` // Milliseconds - KEY METRIC for performance analysis
	CacheLevel       string  `json:"cache_level"`         // "versions", "cache" - cache tier utilization
	SpeedImprovement float64 `json:"speed_improvement"`   // Multiplier vs traditional methods
}

// Commit represents a single commit with enhanced compression information
// Extended with comprehensive compression and caching metadata for performance tracking
type Commit struct {
	Hash       string                 `json:"hash"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Author     string                 `json:"author"`
	FilesCount int                    `json:"files_count"`
	Version    int                    `json:"version"`
	Metadata   map[string]interface{} `json:"metadata"`
	ParentHash string                 `json:"parent_hash,omitempty"`

	// Enhanced compression information for performance analysis
	SnapshotZip     string             `json:"snapshot_zip,omitempty"`     // Legacy field for backward compatibility
	CompressionInfo *CompressionResult `json:"compression_info,omitempty"` // Compression metrics and data
}

// LogManager handles commit history operations with simplified storage system
// Updated to work with simplified 2-tier storage system for optimal performance
type LogManager struct {
	DgitDir    string
	ObjectsDir string
	// Simplified Storage System Integration
	VersionsDir string // 메인 버전 저장소 (.dgit/versions/)
	CommitsDir  string // 커밋 메타데이터 (.dgit/commits/)
	CacheDir    string // 단일 캐시 디렉토리 (.dgit/cache/)
}

// NewLogManager creates a new log manager with simplified storage system
// Initializes with simplified 2-tier storage system integration for optimal performance
func NewLogManager(dgitDir string) *LogManager {
	return &LogManager{
		DgitDir:     dgitDir,
		ObjectsDir:  filepath.Join(dgitDir, "objects"),
		VersionsDir: filepath.Join(dgitDir, "versions"),
		CommitsDir:  filepath.Join(dgitDir, "commits"),
		CacheDir:    filepath.Join(dgitDir, "cache"),
	}
}

// GetCommitHistory returns complete commit history sorted by timestamp (newest first)
// Efficiently loads all commits with compression information
func (lm *LogManager) GetCommitHistory() ([]*Commit, error) {
	entries, err := os.ReadDir(lm.CommitsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read commits directory: %w", err)
	}

	var commits []*Commit
	// Process all commit metadata files
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "v") && strings.HasSuffix(entry.Name(), ".json") {
			commitPath := filepath.Join(lm.CommitsDir, entry.Name())
			commit, err := lm.loadCommit(commitPath)
			if err != nil {
				// Skip failed commits but continue processing others
				continue
			}
			commits = append(commits, commit)
		}
	}

	// Sort commits by timestamp (newest first) for intuitive display
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Timestamp.After(commits[j].Timestamp)
	})

	return commits, nil
}

// GetCommit returns a specific commit by version number
// Efficiently loads individual commit with all metadata
func (lm *LogManager) GetCommit(version int) (*Commit, error) {
	commitPath := filepath.Join(lm.CommitsDir, fmt.Sprintf("v%d.json", version))
	return lm.loadCommit(commitPath)
}

// GetCommitByHash retrieves a commit by its full or short hash
// Supports partial hash matching for user convenience
func (lm *LogManager) GetCommitByHash(hash string) (*Commit, error) {
	entries, err := os.ReadDir(lm.CommitsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read commits directory: %w", err)
	}

	// Search through all commit files for hash match
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "v") && strings.HasSuffix(entry.Name(), ".json") {
			commitPath := filepath.Join(lm.CommitsDir, entry.Name())
			commit, err := lm.loadCommit(commitPath)
			if err != nil {
				continue
			}
			// Support both full and partial hash matching
			if strings.HasPrefix(commit.Hash, hash) {
				return commit, nil
			}
		}
	}
	return nil, fmt.Errorf("commit with hash '%s' not found", hash)
}

// GetCurrentVersion returns the current version number by scanning metadata files
// Efficiently determines the latest version for next commit numbering
func (lm *LogManager) GetCurrentVersion() int {
	entries, err := os.ReadDir(lm.CommitsDir)
	if err != nil {
		return 0
	}

	maxVersion := 0
	// Find the highest version number in commit metadata files
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "v") && strings.HasSuffix(entry.Name(), ".json") {
			versionStr := strings.TrimPrefix(strings.TrimSuffix(entry.Name(), ".json"), "v")
			if version, err := strconv.Atoi(versionStr); err == nil && version > maxVersion {
				maxVersion = version
			}
		}
	}

	return maxVersion
}

// GenerateCommitSummary generates human-readable summary with metrics
// Enhanced to include performance information and cache utilization data
func (lm *LogManager) GenerateCommitSummary(commit *Commit) string {
	summary := fmt.Sprintf("[v%d] %s", commit.Version, commit.Message)

	if commit.FilesCount > 0 {
		summary += fmt.Sprintf(" (%d files)", commit.FilesCount)
	}

	// Add compression information for performance awareness
	if commit.CompressionInfo != nil {
		compressionPercent := (1.0 - commit.CompressionInfo.CompressionRatio) * 100
		switch commit.CompressionInfo.Strategy {
		case "lz4":
			summary += fmt.Sprintf(" • LZ4: %.1f%% (%.1fms)", compressionPercent, commit.CompressionInfo.CompressionTime)
		case "psd_smart":
			summary += fmt.Sprintf(" • Smart PSD: %.1f%% saved", compressionPercent)
		case "design_smart_delta":
			summary += fmt.Sprintf(" • Smart Design: %.1f%% compressed", compressionPercent)
		case "zip":
			summary += fmt.Sprintf(" • ZIP: %.1f%% compressed", compressionPercent)
		case "bsdiff":
			summary += fmt.Sprintf(" • Delta: %.1f%% saved", compressionPercent)
		case "xdelta3":
			summary += fmt.Sprintf(" • XDelta: %.1f%% saved", compressionPercent)
		}

		// Add cache level information for performance context
		if commit.CompressionInfo.CacheLevel != "" {
			summary += fmt.Sprintf(" (%s)", commit.CompressionInfo.CacheLevel)
		}
	}

	// Add design file metadata insights for context
	var insights []string
	for fileName, metadata := range commit.Metadata {
		if metaMap, ok := metadata.(map[string]interface{}); ok {
			if layers, ok := metaMap["layers"].(float64); ok && layers > 0 {
				insights = append(insights, fmt.Sprintf("%s: %.0f layers", filepath.Base(fileName), layers))
			}
		}
	}

	// Include insights if available and not too verbose
	if len(insights) > 0 && len(insights) <= 3 {
		summary += " • " + strings.Join(insights, ", ")
	}

	return summary
}

// GetCompressionStatistics returns compression analytics
// Provides detailed performance metrics across all commits for optimization insights
func (lm *LogManager) GetCompressionStatistics() (*CompressionStatistics, error) {
	commits, err := lm.GetCommitHistory()
	if err != nil {
		return nil, err
	}

	stats := &CompressionStatistics{
		TotalCommits:          len(commits),
		LegacyCommits:         0,
		CompressedCommits:     0,
		TotalSavedSpace:       0,
		StrategyStats:         make(map[string]int),
		CacheLevelStats:       make(map[string]int),
		AvgCompressionTime:    0,
		TotalSpeedImprovement: 0,
	}

	var totalCompressionTime float64
	var totalSpeedImprovement float64
	compressedCount := 0

	// Analyze each commit for statistics
	for _, commit := range commits {
		if commit.CompressionInfo != nil {
			// Track compressed commits with detailed metrics
			stats.CompressedCommits++
			stats.StrategyStats[commit.CompressionInfo.Strategy]++

			// Track cache level utilization for optimization insights
			if commit.CompressionInfo.CacheLevel != "" {
				stats.CacheLevelStats[commit.CompressionInfo.CacheLevel]++
			}

			// Calculate total space savings from compression
			spaceSaved := commit.CompressionInfo.OriginalSize - commit.CompressionInfo.CompressedSize
			stats.TotalSavedSpace += spaceSaved

			// Track performance metrics for continuous improvement
			if commit.CompressionInfo.CompressionTime > 0 {
				totalCompressionTime += commit.CompressionInfo.CompressionTime
				compressedCount++
			}

			if commit.CompressionInfo.SpeedImprovement > 0 {
				totalSpeedImprovement += commit.CompressionInfo.SpeedImprovement
			}
		} else {
			// Track legacy commits for migration planning
			stats.LegacyCommits++
		}
	}

	// Calculate performance averages for insights
	if compressedCount > 0 {
		stats.AvgCompressionTime = totalCompressionTime / float64(compressedCount)
	}
	if stats.CompressedCommits > 0 {
		stats.TotalSpeedImprovement = totalSpeedImprovement / float64(stats.CompressedCommits)
	}

	return stats, nil
}

// CompressionStatistics represents repository performance analytics
// Provides insights into compression system utilization and efficiency
type CompressionStatistics struct {
	TotalCommits          int            `json:"total_commits"`
	LegacyCommits         int            `json:"legacy_commits"`
	CompressedCommits     int            `json:"compressed_commits"`
	TotalSavedSpace       int64          `json:"total_saved_space"`
	StrategyStats         map[string]int `json:"strategy_stats"`
	CacheLevelStats       map[string]int `json:"cache_level_stats"`
	AvgCompressionTime    float64        `json:"avg_compression_time_ms"`
	TotalSpeedImprovement float64        `json:"total_speed_improvement"`
}

// GetCommitStorageInfo returns detailed storage information with metrics
// Enhanced to show cache utilization and performance characteristics
func (lm *LogManager) GetCommitStorageInfo(commit *Commit) string {
	if commit.CompressionInfo == nil {
		// Legacy commit without compression information
		if commit.SnapshotZip != "" {
			return fmt.Sprintf("Legacy ZIP: %s", commit.SnapshotZip)
		}
		return "Unknown storage"
	}

	// Compression system with detailed performance metrics
	switch commit.CompressionInfo.Strategy {
	case "lz4":
		return fmt.Sprintf("LZ4: %s (%.2f MB, %s, %.1fms)",
			commit.CompressionInfo.OutputFile,
			float64(commit.CompressionInfo.CompressedSize)/(1024*1024),
			commit.CompressionInfo.CacheLevel,
			commit.CompressionInfo.CompressionTime)
	case "psd_smart":
		return fmt.Sprintf("Smart PSD Delta: %s (%.2f KB, base: v%d, %.1fms)",
			commit.CompressionInfo.OutputFile,
			float64(commit.CompressionInfo.CompressedSize)/1024,
			commit.CompressionInfo.BaseVersion,
			commit.CompressionInfo.CompressionTime)
	case "design_smart_delta":
		return fmt.Sprintf("Smart Design Delta: %s (%.2f KB, base: v%d)",
			commit.CompressionInfo.OutputFile,
			float64(commit.CompressionInfo.CompressedSize)/1024,
			commit.CompressionInfo.BaseVersion)
	case "zip":
		return fmt.Sprintf("ZIP Snapshot: %s (%.2f MB)",
			commit.CompressionInfo.OutputFile,
			float64(commit.CompressionInfo.CompressedSize)/(1024*1024))
	case "bsdiff":
		return fmt.Sprintf("Binary Delta: %s (%.2f KB, base: v%d)",
			commit.CompressionInfo.OutputFile,
			float64(commit.CompressionInfo.CompressedSize)/1024,
			commit.CompressionInfo.BaseVersion)
	case "xdelta3":
		return fmt.Sprintf("Block Delta: %s (%.2f KB, base: v%d)",
			commit.CompressionInfo.OutputFile,
			float64(commit.CompressionInfo.CompressedSize)/1024,
			commit.CompressionInfo.BaseVersion)
	default:
		return fmt.Sprintf("Unknown: %s", commit.CompressionInfo.OutputFile)
	}
}

// GetCommitEfficiency returns compression efficiency information
// Enhanced with performance metrics and speed improvements
func (lm *LogManager) GetCommitEfficiency(commit *Commit) string {
	if commit.CompressionInfo == nil {
		return "N/A"
	}

	compressionPercent := (1.0 - commit.CompressionInfo.CompressionRatio) * 100

	// Strategy-specific efficiency reporting with performance context
	switch commit.CompressionInfo.Strategy {
	case "lz4":
		speedInfo := ""
		if commit.CompressionInfo.SpeedImprovement > 0 {
			speedInfo = fmt.Sprintf(" (%.1fx faster)", commit.CompressionInfo.SpeedImprovement)
		}
		return fmt.Sprintf("%.1f%% compression%s", compressionPercent, speedInfo)
	case "psd_smart":
		return fmt.Sprintf("%.1f%% space saving (smart delta)", compressionPercent)
	case "design_smart_delta":
		return fmt.Sprintf("%.1f%% compression (smart)", compressionPercent)
	case "zip":
		return fmt.Sprintf("%.1f%% compression", compressionPercent)
	case "bsdiff", "xdelta3":
		return fmt.Sprintf("%.1f%% space saving", compressionPercent)
	default:
		return fmt.Sprintf("%.1f%% efficiency", compressionPercent)
	}
}

// FindCommitsByStorageType finds commits using specific storage strategies
// Enhanced for compression system with strategy filtering
func (lm *LogManager) FindCommitsByStorageType(storageType string) ([]*Commit, error) {
	allCommits, err := lm.GetCommitHistory()
	if err != nil {
		return nil, err
	}

	var filteredCommits []*Commit

	// Filter commits based on storage type with strategy awareness
	for _, commit := range allCommits {
		switch storageType {
		case "legacy":
			// Legacy commits without compression
			if commit.CompressionInfo == nil && commit.SnapshotZip != "" {
				filteredCommits = append(filteredCommits, commit)
			}
		case "fast":
			// Fast compression strategies
			if commit.CompressionInfo != nil &&
				(commit.CompressionInfo.Strategy == "lz4" ||
					commit.CompressionInfo.Strategy == "psd_smart" ||
					commit.CompressionInfo.Strategy == "design_smart_delta") {
				filteredCommits = append(filteredCommits, commit)
			}
		case "lz4":
			// LZ4 compression
			if commit.CompressionInfo != nil && commit.CompressionInfo.Strategy == "lz4" {
				filteredCommits = append(filteredCommits, commit)
			}
		case "smart_delta":
			// Smart delta compression strategies
			if commit.CompressionInfo != nil &&
				(commit.CompressionInfo.Strategy == "psd_smart" ||
					commit.CompressionInfo.Strategy == "design_smart_delta") {
				filteredCommits = append(filteredCommits, commit)
			}
		case "zip":
			// Traditional ZIP compression
			if commit.CompressionInfo != nil && commit.CompressionInfo.Strategy == "zip" {
				filteredCommits = append(filteredCommits, commit)
			}
		case "delta":
			// Binary delta compression strategies
			if commit.CompressionInfo != nil &&
				(commit.CompressionInfo.Strategy == "bsdiff" || commit.CompressionInfo.Strategy == "xdelta3") {
				filteredCommits = append(filteredCommits, commit)
			}
		case "all":
			// All commits regardless of storage type
			filteredCommits = append(filteredCommits, commit)
		}
	}

	return filteredCommits, nil
}

// GetRepositorySizeBreakdown returns detailed size breakdown with cache information
// Enhanced with simplified 2-tier storage system analysis
func (lm *LogManager) GetRepositorySizeBreakdown() (*SizeBreakdown, error) {
	breakdown := &SizeBreakdown{
		ZipFiles:   0,
		DeltaFiles: 0,
		Metadata:   0,
		Versions:   0,
		Cache:      0,
		Total:      0,
	}

	// Calculate traditional objects directory size
	err := filepath.Walk(lm.ObjectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		size := info.Size()
		breakdown.Total += size

		// Categorize files by type for detailed breakdown
		if strings.HasSuffix(path, ".zip") {
			breakdown.ZipFiles += size
		} else if strings.Contains(path, "deltas") {
			breakdown.DeltaFiles += size
		} else if strings.HasSuffix(path, ".json") {
			breakdown.Metadata += size
		}

		return nil
	})

	if err != nil {
		return breakdown, err
	}

	// Calculate simplified storage sizes
	lm.calculateDirectorySize(lm.VersionsDir, &breakdown.Versions)
	lm.calculateDirectorySize(lm.CacheDir, &breakdown.Cache)

	// Include storage sizes in total for complete picture
	breakdown.Total += breakdown.Versions + breakdown.Cache

	return breakdown, nil
}

// calculateDirectorySize calculates total size of a directory recursively
// Helper function for storage utilization analysis
func (lm *LogManager) calculateDirectorySize(dir string, size *int64) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			*size += info.Size()
		}
		return nil
	})
}

// SizeBreakdown represents repository size analysis
// Enhanced with simplified storage information for complete storage visibility
type SizeBreakdown struct {
	ZipFiles   int64 `json:"zip_files"`   // Traditional ZIP snapshots
	DeltaFiles int64 `json:"delta_files"` // Delta compression files
	Metadata   int64 `json:"metadata"`    // Commit metadata JSON files
	Versions   int64 `json:"versions"`    // Versions directory (.dgit/versions/)
	Cache      int64 `json:"cache"`       // Cache directory (.dgit/cache/)
	Total      int64 `json:"total"`       // Total repository size including all storage
}

// GetCacheUtilization returns cache utilization statistics
// Provides insights into simplified storage system performance and efficiency
func (lm *LogManager) GetCacheUtilization() (*CacheUtilization, error) {
	commits, err := lm.GetCommitHistory()
	if err != nil {
		return nil, err
	}

	utilization := &CacheUtilization{
		VersionsFiles:  0,
		CacheFiles:     0,
		TotalCacheSize: 0,
	}

	// Analyze cache utilization across all commits
	for _, commit := range commits {
		if commit.CompressionInfo != nil {
			// Track cache tier utilization for optimization insights
			switch commit.CompressionInfo.CacheLevel {
			case "versions":
				utilization.VersionsFiles++
			case "cache":
				utilization.CacheFiles++
			}
			utilization.TotalCacheSize += commit.CompressionInfo.CompressedSize
		}
	}

	return utilization, nil
}

// CacheUtilization represents detailed cache usage statistics
// Provides insights for optimizing simplified storage system performance
type CacheUtilization struct {
	VersionsFiles  int   `json:"versions_files"`   // Files in versions directory
	CacheFiles     int   `json:"cache_files"`      // Files in cache directory
	TotalCacheSize int64 `json:"total_cache_size"` // Total cached data size
}

// loadCommit loads a commit from a JSON metadata file
// Core function for reading commit information with error handling
func (lm *LogManager) loadCommit(path string) (*Commit, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var commit Commit
	if err := json.Unmarshal(data, &commit); err != nil {
		return nil, err
	}

	return &commit, nil
}
