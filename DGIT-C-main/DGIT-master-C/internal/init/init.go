package init

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DGitDir defines the standard DGit repository directory name
const DGitDir = ".dgit"

// RepositoryInitializer handles repository initialization
type RepositoryInitializer struct{}

// NewRepositoryInitializer creates a new repository initializer instance
func NewRepositoryInitializer() *RepositoryInitializer {
	return &RepositoryInitializer{}
}

// RepositoryConfig represents repository configuration
type RepositoryConfig struct {
	Author      string    `json:"author"`
	Email       string    `json:"email"`
	Created     time.Time `json:"created"`
	Version     string    `json:"version"`
	Description string    `json:"description"`

	// Compression System Configuration
	Compression CompressionConfig `json:"compression"`

	// Performance Monitoring Settings
	Performance PerformanceConfig `json:"performance"`
}

// CompressionConfig represents simplified compression settings
type CompressionConfig struct {
	// LZ4 Fast Compression
	LZ4Config LZ4StageConfig `json:"lz4_stage"`

	// Background Optimization (Optional)
	ZstdConfig ZstdStageConfig `json:"zstd_stage"`

	// Long-term Storage (Optional)
	ArchiveConfig ArchiveStageConfig `json:"archive_stage"`

	// Cache Management Settings
	CacheConfig SmartCacheConfig `json:"cache"`
}

// LZ4StageConfig configures fast compression
type LZ4StageConfig struct {
	Enabled          bool  `json:"enabled"`           // Enable LZ4 compression
	MaxFileSize      int64 `json:"max_file_size"`     // Max file size for LZ4 (bytes)
	CompressionLevel int   `json:"compression_level"` // LZ4 compression level (1-9)
	CacheRetention   int   `json:"cache_retention"`   // Hours to keep in cache
}

// ZstdStageConfig configures background optimization
type ZstdStageConfig struct {
	Enabled          bool    `json:"enabled"`           // Enable background Zstd optimization
	CompressionLevel int     `json:"compression_level"` // Zstd level (1-22, 3=balanced)
	OptimizeInterval int     `json:"optimize_interval"` // Minutes between optimization runs
	MinIdleTime      int     `json:"min_idle_time"`     // Seconds of idle time before optimization
	CompressionRatio float64 `json:"compression_ratio"` // Target compression ratio
}

// ArchiveStageConfig configures long-term storage
type ArchiveStageConfig struct {
	Enabled          bool  `json:"enabled"`            // Enable archival compression
	CompressionLevel int   `json:"compression_level"`  // Zstd level 22 (maximum compression)
	ArchiveAfterDays int   `json:"archive_after_days"` // Days before moving to archive
	MaxArchiveSize   int64 `json:"max_archive_size"`   // Max size per archive file (bytes)
}

// SmartCacheConfig configures cache management
type SmartCacheConfig struct {
	MainCacheSize   int64  `json:"main_cache_size"`   // Max main cache size (MB)
	BackupCacheSize int64  `json:"backup_cache_size"` // Max backup cache size (MB)
	AccessThreshold int    `json:"access_threshold"`  // Accesses needed to promote cache
	EvictionPolicy  string `json:"eviction_policy"`   // "LRU", "LFU", "FIFO"
}

// PerformanceConfig configures monitoring systems
type PerformanceConfig struct {
	EnableMetrics      bool `json:"enable_metrics"`       // Collect performance metrics
	LogCompressionTime bool `json:"log_compression_time"` // Log compression timing data
	LogCacheHits       bool `json:"log_cache_hits"`       // Log cache hit/miss ratios
	StatsRetentionDays int  `json:"stats_retention_days"` // Days to keep performance statistics
}

// InitializeRepository initializes a new DGit repository
func (ri *RepositoryInitializer) InitializeRepository(path string) error {
	dgitPath := filepath.Join(path, DGitDir)

	if _, err := os.Stat(dgitPath); !os.IsNotExist(err) {
		return fmt.Errorf("DGit repository already exists in %s", path)
	}

	if err := ri.createStructure(dgitPath); err != nil {
		return fmt.Errorf("failed to create DGit structure: %w", err)
	}

	if err := ri.createConfig(dgitPath); err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}

	if err := ri.createPerformanceMonitoring(dgitPath); err != nil {
		return fmt.Errorf("failed to create performance monitoring: %w", err)
	}

	if err := ri.createInitialHead(dgitPath); err != nil {
		return fmt.Errorf("failed to create HEAD file: %w", err)
	}

	return nil
}

// createStructure creates simplified directory structure
func (ri *RepositoryInitializer) createStructure(dgitPath string) error {
	if err := os.MkdirAll(dgitPath, 0755); err != nil {
		return err
	}

	// Simplified Structure
	subdirs := []string{
		// Main Storage Directories
		"versions",       // Primary version storage
		"commits",        // Commit metadata
		"cache",          // Single cache directory
		"cache/temp",     // Temporary compression files
		"cache/metadata", // Cache metadata

		// Working Areas
		"staging",

		// System Directories
		"temp",  // Temporary workspace
		"refs",  // Reference information
		"hooks", // Hook scripts

		// Performance Monitoring
		"logs",
		"metrics",
	}

	for _, subdir := range subdirs {
		subdirPath := filepath.Join(dgitPath, subdir)
		if err := os.MkdirAll(subdirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", subdirPath, err)
		}
	}

	if err := ri.createCacheIndexes(dgitPath); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes creates lookup indexes
func (ri *RepositoryInitializer) createCacheIndexes(dgitPath string) error {
	indexes := map[string]interface{}{
		"cache/metadata/index.json": make(map[string]interface{}),
		"versions/index.json":       make(map[string]interface{}),
		"commits/index.json":        make(map[string]interface{}),
	}

	for indexPath, indexData := range indexes {
		fullPath := filepath.Join(dgitPath, indexPath)
		data, err := json.MarshalIndent(indexData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal index %s: %w", indexPath, err)
		}

		if err := os.WriteFile(fullPath, data, 0644); err != nil {
			return fmt.Errorf("failed to create index %s: %w", indexPath, err)
		}
	}

	return nil
}

// createConfig creates simplified configuration
func (ri *RepositoryInitializer) createConfig(dgitPath string) error {
	config := RepositoryConfig{
		Author:      "DGit User",
		Email:       "user@dgit.local",
		Created:     time.Now(),
		Version:     "2.0.0",
		Description: "DGit repository with simplified structure",

		// Simplified Compression Configuration
		Compression: CompressionConfig{
			// LZ4 Fast Compression (single compression method)
			LZ4Config: LZ4StageConfig{
				Enabled:          true,
				MaxFileSize:      500 * 1024 * 1024, // 500MB max
				CompressionLevel: 1,                 // Fastest LZ4 level
				CacheRetention:   24,                // Keep 24 hours
			},

			// Background Optimization (Disabled for simplicity)
			ZstdConfig: ZstdStageConfig{
				Enabled:          false,
				CompressionLevel: 3,
				OptimizeInterval: 60,  // 1 hour
				MinIdleTime:      300, // 5 minutes wait
				CompressionRatio: 0.4,
			},

			// Archive (Disabled for simplicity)
			ArchiveConfig: ArchiveStageConfig{
				Enabled:          false,
				CompressionLevel: 22,
				ArchiveAfterDays: 90,                     // 3 months later
				MaxArchiveSize:   5 * 1024 * 1024 * 1024, // 5GB
			},

			// Simplified Cache Configuration
			CacheConfig: SmartCacheConfig{
				MainCacheSize:   1 * 1024, // 1GB single cache
				BackupCacheSize: 0,        // Not used
				AccessThreshold: 1,        // Immediate cache
				EvictionPolicy:  "LRU",
			},
		},

		// Performance Monitoring Configuration
		Performance: PerformanceConfig{
			EnableMetrics:      true,
			LogCompressionTime: true,
			LogCacheHits:       false, // Simplified
			StatsRetentionDays: 30,    // 1 month
		},
	}

	configPath := filepath.Join(dgitPath, "config")
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// createPerformanceMonitoring sets up performance tracking
func (ri *RepositoryInitializer) createPerformanceMonitoring(dgitPath string) error {
	perfSummary := map[string]interface{}{
		"created_at":    time.Now(),
		"version":       "2.0.0",
		"total_commits": 0,
		"total_files":   0,
		"cache_stats": map[string]int{
			"versions_hits": 0, // Versions directory hits
			"cache_hits":    0, // Cache directory hits
			"misses":        0, // Cache misses
		},
		"compression_stats": map[string]float64{
			"avg_lz4_time":          0.0, // Average LZ4 compression time
			"avg_zstd_time":         0.0, // Average Zstd compression time
			"avg_compression_ratio": 0.0, // Average compression efficiency
		},
	}

	perfPath := filepath.Join(dgitPath, "metrics", "summary.json")
	perfData, err := json.MarshalIndent(perfSummary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal performance summary: %w", err)
	}

	if err := os.WriteFile(perfPath, perfData, 0644); err != nil {
		return fmt.Errorf("failed to create performance summary: %w", err)
	}

	logFiles := []string{
		"logs/compression/lz4.log",
		"logs/compression/zstd.log",
		"logs/cache/hits.log",
		"logs/cache/evictions.log",
		"logs/performance.log",
	}

	for _, logFile := range logFiles {
		logPath := filepath.Join(dgitPath, logFile)

		// Create directory for log file if it doesn't exist
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
		}

		initialLog := fmt.Sprintf("# DGit Log - %s\n# Created: %s\n\n",
			filepath.Base(logFile), time.Now().Format(time.RFC3339))

		if err := os.WriteFile(logPath, []byte(initialLog), 0644); err != nil {
			return fmt.Errorf("failed to create log file %s: %w", logFile, err)
		}
	}

	return nil
}

// createInitialHead creates the initial HEAD file
func (ri *RepositoryInitializer) createInitialHead(dgitPath string) error {
	headPath := filepath.Join(dgitPath, "HEAD")
	if err := os.WriteFile(headPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to create HEAD file: %w", err)
	}
	return nil
}

// IsDGitRepository checks if a path contains a valid DGit repository
func IsDGitRepository(path string) bool {
	dgitPath := filepath.Join(path, DGitDir)
	info, err := os.Stat(dgitPath)
	if err != nil || !info.IsDir() {
		return false
	}

	// Check for essential directories in new structure
	versionsPath := filepath.Join(dgitPath, "versions")
	if info, err := os.Stat(versionsPath); err != nil || !info.IsDir() {
		return false
	}

	return true
}

// GetConfig loads repository configuration
func GetConfig(dgitPath string) (*RepositoryConfig, error) {
	configPath := filepath.Join(dgitPath, "config")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config RepositoryConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// UpdateConfig saves repository configuration
func UpdateConfig(dgitPath string, config *RepositoryConfig) error {
	configPath := filepath.Join(dgitPath, "config")

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
