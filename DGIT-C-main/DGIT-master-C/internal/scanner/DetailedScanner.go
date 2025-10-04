// =============================================================================
// 4. 상세 스캐너 (scanner/detailed_scanner.go)
// =============================================================================

package scanner

import (
	"dgit/internal/scanner/illustrator"
	"dgit/internal/scanner/photoshop"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetailedFileInfo contains comprehensive file analysis results
type DetailedFileInfo struct {
	Path       string
	Type       string
	FileSize   int64
	Dimensions string
	ColorMode  string
	Version    string
	Layers     int
	Artboards  int
	Objects    int
	LayerNames []string
}

// DetailedScanner performs comprehensive file analysis
type DetailedScanner struct{}

// NewDetailedScanner creates a new detailed scanner instance
func NewDetailedScanner() *DetailedScanner {
	return &DetailedScanner{}
}

// AnalyzeFile performs comprehensive analysis of a single design file
func (ds *DetailedScanner) AnalyzeFile(filePath string) (*DetailedFileInfo, error) {
	if !IsDesignFile(filePath) {
		return nil, fmt.Errorf("unsupported file type")
	}

	// fileName 변수 제거 (사용되지 않음)
	fileType := strings.ToLower(filepath.Ext(filePath)[1:])

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	result := &DetailedFileInfo{
		Path:       filePath,
		Type:       fileType,
		FileSize:   fileInfo.Size(),
		Dimensions: "Unknown",
		ColorMode:  "Unknown",
		Version:    "Unknown",
		Layers:     0,
		Artboards:  1,
		Objects:    0,
		LayerNames: []string{},
	}

	// Perform type-specific analysis
	switch fileType {
	case "ai":
		return ds.analyzeAI(filePath, result)
	case "psd":
		return ds.analyzePSD(filePath, result)
	case "sketch":
		return ds.analyzeSketch(filePath, result)
	case "fig":
		return ds.analyzeFigma(filePath, result)
	case "xd":
		return ds.analyzeXD(filePath, result)
	default:
		return result, nil
	}
}

// analyzeAI performs detailed Adobe Illustrator file analysis
func (ds *DetailedScanner) analyzeAI(filePath string, result *DetailedFileInfo) (*DetailedFileInfo, error) {
	aiInfo, err := illustrator.GetAIInfo(filePath)
	if err != nil {
		return result, err
	}

	result.Dimensions = fmt.Sprintf("%dx%d px", aiInfo.Width, aiInfo.Height)
	result.ColorMode = aiInfo.ColorMode
	result.Version = aiInfo.Version
	result.Layers = aiInfo.LayerCount
	result.Artboards = aiInfo.ArtboardCount
	result.Objects = aiInfo.ObjectCount
	result.LayerNames = aiInfo.LayerNames

	return result, nil
}

// analyzePSD performs detailed Adobe Photoshop file analysis
func (ds *DetailedScanner) analyzePSD(filePath string, result *DetailedFileInfo) (*DetailedFileInfo, error) {
	psdInfo, err := photoshop.GetPSDInfo(filePath)
	if err != nil {
		return result, err
	}

	result.Dimensions = fmt.Sprintf("%dx%d px", psdInfo.Width, psdInfo.Height)
	result.ColorMode = mapPSDColorMode(psdInfo.Channels, psdInfo.Bits)
	result.Version = "Adobe Photoshop"
	result.Layers = psdInfo.LayerCount
	result.LayerNames = psdInfo.LayerNames
	result.Objects = len(psdInfo.LayerNames) * 2

	return result, nil
}

// analyzeSketch provides basic Sketch file information
func (ds *DetailedScanner) analyzeSketch(filePath string, result *DetailedFileInfo) (*DetailedFileInfo, error) {
	result.ColorMode = "RGB"
	result.Version = "Sketch App"
	result.Layers = 1
	result.LayerNames = []string{"Sketch Layer"}
	return result, nil
}

// analyzeFigma provides basic Figma file information
func (ds *DetailedScanner) analyzeFigma(filePath string, result *DetailedFileInfo) (*DetailedFileInfo, error) {
	result.ColorMode = "RGB"
	result.Version = "Figma"
	result.Layers = 1
	result.LayerNames = []string{"Figma Frame"}
	return result, nil
}

// analyzeXD provides basic Adobe XD file information
func (ds *DetailedScanner) analyzeXD(filePath string, result *DetailedFileInfo) (*DetailedFileInfo, error) {
	result.ColorMode = "RGB"
	result.Version = "Adobe XD"
	result.Layers = 1
	result.LayerNames = []string{"XD Artboard"}
	return result, nil
}

// mapPSDColorMode maps PSD channel information to readable color mode names
func mapPSDColorMode(channels, bits int) string {
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
