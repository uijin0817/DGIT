package illustrator

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

// AIInfo contains comprehensive metadata extracted from Adobe Illustrator files
// Provides detailed information about AI file structure, layers, and design elements
type AIInfo struct {
	Width          int      // Canvas width in points
	Height         int      // Canvas height in points
	ColorMode      string   // Color mode: RGB, CMYK, Grayscale
	Version        string   // Adobe Illustrator version (e.g., "CC 2025 (29.x)")
	LayerCount     int      // Total number of layers in the document
	LayerNames     []string // Names of all layers
	ArtboardCount  int      // Number of artboards/pages
	ObjectCount    int      // Estimated number of design objects
	FontCount      int      // Number of unique fonts used
	EmbeddedImages int      // Number of embedded images
}

// GetAIInfo extracts comprehensive metadata from Adobe Illustrator files
// Analyzes AI file structure and returns detailed design information
func GetAIInfo(filePath string) (*AIInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open AI file: %w", err)
	}
	defer file.Close()

	// Initialize AI info structure with default values
	aiInfo := &AIInfo{
		ColorMode:      "RGB",        // Default color mode
		Version:        "Unknown",    // Will be extracted from file
		LayerNames:     []string{},   // Will be populated from file
		ArtboardCount:  1,           // Default single artboard
		LayerCount:     0,           // Will be counted from file
		ObjectCount:    0,           // Will be estimated from file
		FontCount:      0,           // Will be counted from file
		EmbeddedImages: 0,           // Will be counted from file
	}

	// Scan file content efficiently (first 1000 lines for performance)
	scanner := bufio.NewScanner(file)
	var content strings.Builder
	lineCount := 0
	const maxLines = 1000 // Increased from original for better metadata extraction

	for scanner.Scan() && lineCount < maxLines {
		line := scanner.Text()
		content.WriteString(line)
		content.WriteString("\n")
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	fileContent := content.String()

	// Extract comprehensive metadata from file content
	// 1. Extract Adobe Illustrator version information
	if version := extractCreatorVersion(fileContent); version != "" {
		aiInfo.Version = version
	}

	// 2. Extract canvas dimensions from MediaBox
	if width, height := extractMediaBox(fileContent); width > 0 && height > 0 {
		aiInfo.Width = width
		aiInfo.Height = height
	}

	// 3. Extract layer information and names
	layerNames := extractLayerNames(fileContent)
	aiInfo.LayerNames = layerNames
	aiInfo.LayerCount = len(layerNames)

	// Ensure at least one default layer if none found
	if aiInfo.LayerCount == 0 {
		aiInfo.LayerCount = 1
		aiInfo.LayerNames = []string{"Layer 1"}
	}

	// 4. Extract artboard/page count
	if pages := extractPageCount(fileContent); pages > 0 {
		aiInfo.ArtboardCount = pages
	}

	// 5. Determine color mode from document content
	if colorMode := extractColorMode(fileContent); colorMode != "" {
		aiInfo.ColorMode = colorMode
	}

	// 6. Estimate number of design objects
	aiInfo.ObjectCount = countObjects(fileContent)

	// 7. Count unique fonts used in the document
	aiInfo.FontCount = countFonts(fileContent)

	// 8. Count embedded/linked images
	aiInfo.EmbeddedImages = countImages(fileContent)

	return aiInfo, nil
}

// extractCreatorVersion extracts Adobe Illustrator version from file metadata
// Supports multiple metadata formats and version detection methods
func extractCreatorVersion(content string) string {
	// Strategy 1: XMP metadata CreatorTool extraction (most accurate)
	creatorToolRe := regexp.MustCompile(`<xmp:CreatorTool>([^<]+)</xmp:CreatorTool>`)
	if matches := creatorToolRe.FindStringSubmatch(content); len(matches) > 1 {
		creatorTool := matches[1]
		// Extract version from "Adobe Illustrator 29.6 (Macintosh)" format
		versionRe := regexp.MustCompile(`Adobe Illustrator (\d+\.?\d*)`)
		if versionMatches := versionRe.FindStringSubmatch(creatorTool); len(versionMatches) > 1 {
			version := versionMatches[1]
			return mapVersionToName(version)
		}
		return creatorTool
	}

	// Strategy 2: Software agent metadata extraction
	agentRe := regexp.MustCompile(`<stEvt:softwareAgent>Adobe Illustrator ([^<]+)</stEvt:softwareAgent>`)
	if agentMatches := agentRe.FindStringSubmatch(content); len(agentMatches) > 1 {
		return "Adobe Illustrator " + agentMatches[1]
	}

	// Strategy 3: Legacy illustrator:CreatorVersion pattern
	re := regexp.MustCompile(`illustrator:CreatorVersion="([^"]+)"`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		version := matches[1]
		return mapVersionToName(version)
	}

	// Strategy 4: Creator information extraction (more specific pattern)
	creatorRe := regexp.MustCompile(`/Creator\s*\(([^)]*Adobe Illustrator[^)]*)\)`)
	if creatorMatches := creatorRe.FindStringSubmatch(content); len(creatorMatches) > 1 {
		creatorInfo := creatorMatches[1]
		// Extract version number from creator info
		versionRe := regexp.MustCompile(`(\d+\.\d+)`)
		if versionMatches := versionRe.FindStringSubmatch(creatorInfo); len(versionMatches) > 1 {
			return mapVersionToName(versionMatches[1])
		}
		return "Adobe Illustrator"
	}

	// Strategy 5: Producer information extraction
	producerRe := regexp.MustCompile(`/Producer\s*\(([^)]*Adobe[^)]*)\)`)
	if producerMatches := producerRe.FindStringSubmatch(content); len(producerMatches) > 1 {
		return "Adobe Illustrator"
	}

	// Strategy 6: PDF version inference (fallback)
	pdfVersionRe := regexp.MustCompile(`%PDF-(\d+\.\d+)`)
	if pdfMatches := pdfVersionRe.FindStringSubmatch(content); len(pdfMatches) > 1 {
		return "Adobe Illustrator (PDF " + pdfMatches[1] + ")"
	}

	return ""
}

// mapVersionToName maps version numbers to user-friendly product names
// Provides clear identification of Adobe Illustrator versions
func mapVersionToName(version string) string {
	switch {
	case strings.HasPrefix(version, "29."):
		return "CC 2025 (29.x)"
	case strings.HasPrefix(version, "28."):
		return "CC 2024 (28.x)"
	case strings.HasPrefix(version, "27."):
		return "CC 2023 (27.x)"
	case strings.HasPrefix(version, "26."):
		return "CC 2022 (26.x)"
	case strings.HasPrefix(version, "25."):
		return "CC 2021 (25.x)"
	case strings.HasPrefix(version, "24."):
		return "CC 2020 (24.x)"
	case strings.HasPrefix(version, "23."):
		return "CC 2019 (23.x)"
	case strings.HasPrefix(version, "22."):
		return "CC 2018 (22.x)"
	case strings.HasPrefix(version, "21."):
		return "CC 2017 (21.x)"
	case strings.HasPrefix(version, "20."):
		return "CC 2015 (20.x)"
	case strings.HasPrefix(version, "19."):
		return "CC 2015 (19.x)"
	case strings.HasPrefix(version, "18."):
		return "CC 2014 (18.x)"
	case strings.HasPrefix(version, "17."):
		return "CC (17.x)"
	case strings.HasPrefix(version, "16."):
		return "CS6 (16.x)"
	default:
		return "Adobe Illustrator " + version
	}
}

// extractMediaBox extracts canvas dimensions from PDF MediaBox specification
// Returns width and height in points (standard PDF measurement unit)
func extractMediaBox(content string) (int, int) {
	// MediaBox format: /MediaBox [x1 y1 x2 y2]
	re := regexp.MustCompile(`/MediaBox\s*\[\s*([0-9.-]+)\s+([0-9.-]+)\s+([0-9.-]+)\s+([0-9.-]+)\s*\]`)
	matches := re.FindStringSubmatch(content)
	if len(matches) >= 5 {
		x1, err1 := strconv.ParseFloat(matches[1], 64)
		y1, err2 := strconv.ParseFloat(matches[2], 64)
		x2, err3 := strconv.ParseFloat(matches[3], 64)
		y2, err4 := strconv.ParseFloat(matches[4], 64)
		
		if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
			// Calculate width and height from coordinates
			return int(x2 - x1), int(y2 - y1)
		}
	}
	
	// Return default A4 size if MediaBox not found or invalid
	return 595, 842
}

// extractLayerNames extracts layer names using multiple detection strategies
// Handles various layer naming formats and encoding methods
func extractLayerNames(content string) []string {
	var layerNames []string
	seenNames := make(map[string]bool)

	// Strategy 1: OCG (Optional Content Group) reference counting
	// Extract layer count from OCG reference arrays
	ocgArrayRe := regexp.MustCompile(`/OCGs\[([^\]]+)\]`)
	if ocgMatches := ocgArrayRe.FindStringSubmatch(content); len(ocgMatches) > 1 {
		// Parse "38 0 R 39 0 R 40 0 R 41 0 R 42 0 R" format
		refs := strings.Fields(ocgMatches[1])
		refCount := 0
		for i := 0; i < len(refs); i += 3 { // "38 0 R" format, so increment by 3
			if i+2 < len(refs) && refs[i+2] == "R" {
				refCount++
			}
		}
		
		// Generate numbered layer names when exact names can't be extracted
		if refCount > 0 {
			for i := 1; i <= refCount; i++ {
				layerName := fmt.Sprintf("레이어 %d", i)
				layerNames = append(layerNames, layerName)
				seenNames[layerName] = true
			}
		}
	}

	// Strategy 2: Standard string layer names (OCG Name patterns)
	nameRe := regexp.MustCompile(`/Name\s*\(([^)]+)\)`)
	nameMatches := nameRe.FindAllStringSubmatch(content, -1)
	for _, match := range nameMatches {
		if len(match) > 1 {
			layerName := strings.TrimSpace(match[1])
			// Filter out system layers and duplicates
			if layerName != "" && !isSystemLayer(layerName) && !seenNames[layerName] {
				seenNames[layerName] = true
				layerNames = append(layerNames, layerName)
			}
		}
	}

	// Strategy 3: Hex-encoded layer names (UTF-16 format)
	hexRe := regexp.MustCompile(`/Name\s*<([A-Fa-f0-9]+)>`)
	hexMatches := hexRe.FindAllStringSubmatch(content, -1)
	for _, match := range hexMatches {
		if len(match) > 1 {
			decoded, err := decodeUTF16HexString(match[1])
			if err == nil && decoded != "" && !isSystemLayer(decoded) && !seenNames[decoded] {
				seenNames[decoded] = true
				layerNames = append(layerNames, decoded)
			}
		}
	}

	// Strategy 4: AI-specific layer patterns
	aiLayerRe := regexp.MustCompile(`/AI([0-9]+)_Layer\s*\(([^)]+)\)`)
	aiLayerMatches := aiLayerRe.FindAllStringSubmatch(content, -1)
	for _, match := range aiLayerMatches {
		if len(match) > 2 {
			layerName := strings.TrimSpace(match[2])
			if layerName != "" && !isSystemLayer(layerName) && !seenNames[layerName] {
				seenNames[layerName] = true
				layerNames = append(layerNames, layerName)
			}
		}
	}

	return layerNames
}

// isSystemLayer identifies system-generated or technical layers
// Filters out layers that are not user-created content layers
func isSystemLayer(name string) bool {
	systemLayers := []string{
		"View", "Print", "Guides", "Grid", "Rulers", 
		"Template", "Locked", "Hidden", "Outline",
		"Preview", "Overprint", "Transparency",
	}
	
	// Check if layer name contains any system layer keywords
	for _, system := range systemLayers {
		if strings.Contains(name, system) {
			return true
		}
	}
	
	return false
}

// extractPageCount extracts the number of pages/artboards from file metadata
// Supports multiple page counting methods for comprehensive detection
func extractPageCount(content string) int {
	// Strategy 1: XMP metadata page count (most reliable)
	nPagesRe := regexp.MustCompile(`<xmpTPg:NPages>(\d+)</xmpTPg:NPages>`)
	if matches := nPagesRe.FindStringSubmatch(content); len(matches) > 1 {
		if count, err := strconv.Atoi(matches[1]); err == nil && count > 0 {
			return count
		}
	}
	
	// Strategy 2: PDF Count pattern
	re := regexp.MustCompile(`/Count\s+(\d+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		if count, err := strconv.Atoi(matches[1]); err == nil && count > 0 {
			return count
		}
	}
	
	// Default to single page if no count found
	return 1
}

// extractColorMode determines the color mode used in the document
// Analyzes color space definitions to identify RGB, CMYK, or Grayscale
func extractColorMode(content string) string {
	// Check for CMYK color mode indicators
	if strings.Contains(content, "/DeviceCMYK") || 
	   strings.Contains(content, "CMYK") ||
	   strings.Contains(content, "/Separation") {
		return "CMYK"
	}
	
	// Check for Grayscale color mode indicators
	if strings.Contains(content, "/DeviceGray") || strings.Contains(content, "Gray") {
		return "Grayscale"
	}
	
	// Default to RGB color mode
	return "RGB"
}

// countObjects estimates the number of design objects in the document
// Counts PDF objects as a proxy for design complexity
func countObjects(content string) int {
	// Count PDF object definitions (pattern: "number number obj")
	re := regexp.MustCompile(`\d+\s+\d+\s+obj`)
	matches := re.FindAllString(content, -1)
	count := len(matches)
	
	// Return reasonable default if no objects found
	if count == 0 {
		return 10
	}
	return count
}

// countFonts counts the number of unique fonts used in the document
// Analyzes font definitions to determine typography diversity
func countFonts(content string) int {
	fontMap := make(map[string]bool)
	
	// Strategy 1: BaseFont pattern extraction
	baseFontRe := regexp.MustCompile(`/BaseFont\s*/([^/\s\]]+)`)
	baseFontMatches := baseFontRe.FindAllStringSubmatch(content, -1)
	for _, match := range baseFontMatches {
		if len(match) > 1 {
			fontMap[match[1]] = true
		}
	}
	
	// Strategy 2: FontName pattern extraction
	fontNameRe := regexp.MustCompile(`/FontName\s*/([^/\s\]]+)`)
	fontNameMatches := fontNameRe.FindAllStringSubmatch(content, -1)
	for _, match := range fontNameMatches {
		if len(match) > 1 {
			fontMap[match[1]] = true
		}
	}
	
	// Return count of unique fonts found
	return len(fontMap)
}

// countImages estimates the number of images in the document
// Counts various image format references and filters
func countImages(content string) int {
	imageCount := 0
	
	// Search for various image-related patterns
	patterns := []string{
		`/Subtype\s*/Image`,        // Generic image subtype
		`/Filter\s*/DCTDecode`,     // JPEG compression
		`/Filter\s*/JPXDecode`,     // JPEG 2000 compression
		`/Filter\s*/CCITTFaxDecode`, // TIFF/Fax compression
	}
	
	// Count matches for each image pattern
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(content, -1)
		imageCount += len(matches)
	}
	
	return imageCount
}

// decodeUTF16HexString decodes UTF-16 hex-encoded strings from AI files
// Handles Unicode layer names and text content encoding
func decodeUTF16HexString(hexStr string) (string, error) {
	// Remove BOM (Byte Order Mark) if present
	if strings.HasPrefix(hexStr, "FEFF") {
		hexStr = hexStr[4:]
	}

	// Validate hex string length (must be even)
	if len(hexStr)%2 != 0 {
		return "", fmt.Errorf("invalid hex string length")
	}

	if len(hexStr) == 0 {
		return "", fmt.Errorf("empty hex string")
	}

	// Convert hex string to byte array
	byteData := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr)/2; i++ {
		byteVal, err := strconv.ParseUint(hexStr[i*2:i*2+2], 16, 8)
		if err != nil {
			return "", err
		}
		byteData[i] = byte(byteVal)
	}

	// Validate UTF-16 byte length (must be even)
	if len(byteData)%2 != 0 {
		return "", fmt.Errorf("invalid UTF-16 byte length")
	}

	// Convert bytes to UTF-16 code units (big-endian)
	u16 := make([]uint16, len(byteData)/2)
	for i := 0; i < len(byteData)/2; i++ {
		u16[i] = uint16(byteData[i*2])<<8 | uint16(byteData[i*2+1])
	}

	// Decode UTF-16 to UTF-8 string
	decoded := string(utf16.Decode(u16))
	
	// Validate decoded result
	if len(decoded) == 0 {
		return "", fmt.Errorf("empty decoded string")
	}

	return decoded, nil
}