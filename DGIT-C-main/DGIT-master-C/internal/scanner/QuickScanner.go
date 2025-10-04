package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// QuickScanResult contains basic scan results without heavy analysis
type QuickScanResult struct {
	TotalFiles int
	TotalSize  int64
	TypeCounts map[string]int
	ScanTime   time.Duration
	Files      []BasicFileInfo
}

// BasicFileInfo contains minimal file information for quick scanning
type BasicFileInfo struct {
	Path string
	Type string
	Size int64
}

// QuickScanner performs lightweight file discovery
type QuickScanner struct{}

// NewQuickScanner creates a new quick scanner instance
func NewQuickScanner() *QuickScanner {
	return &QuickScanner{}
}

// Scan performs quick directory scanning without metadata extraction
func (qs *QuickScanner) Scan(folderPath string) (*QuickScanResult, error) {
	startTime := time.Now()

	result := &QuickScanResult{
		TypeCounts: make(map[string]int),
		Files:      []BasicFileInfo{},
	}

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue scanning despite errors
		}

		if info.IsDir() {
			// Skip system directories
			if info.Name() == ".git" || info.Name() == ".dgit" {
				return filepath.SkipDir
			}
			return nil
		}

		if IsDesignFile(path) {
			fileType := strings.ToLower(filepath.Ext(path)[1:])

			result.TotalFiles++
			result.TotalSize += info.Size()
			result.TypeCounts[fileType]++

			result.Files = append(result.Files, BasicFileInfo{
				Path: path,
				Type: fileType,
				Size: info.Size(),
			})
		}
		return nil
	})

	result.ScanTime = time.Since(startTime)
	return result, err
}
