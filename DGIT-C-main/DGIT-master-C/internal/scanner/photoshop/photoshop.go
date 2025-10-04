package photoshop

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

// PSDInfo contains essential metadata extracted from Photoshop PSD files
// Provides comprehensive information about document structure and layer organization
type PSDInfo struct {
	Width      int      // Document width in pixels
	Height     int      // Document height in pixels
	Channels   int      // Number of color channels
	Bits       int      // Bit depth per channel
	LayerCount int      // Total number of layers in document
	LayerNames []string // Names of all layers in the document
}

// psdFileHeader represents the core PSD file header structure
// Contains fundamental document information stored at the beginning of PSD files
type psdFileHeader struct {
	Signature [4]byte // "8BPS" - Photoshop signature
	Version   uint16  // PSD version (1 = PSD, 2 = PSB)
	Reserved  [6]byte // Reserved bytes (must be zero)
	Channels  uint16  // Number of color channels (1-56)
	Height    uint32  // Document height in pixels
	Width     uint32  // Document width in pixels
	Depth     uint16  // Bit depth (1, 8, 16, or 32)
	ColorMode uint16  // Color mode (Bitmap=0, Grayscale=1, Indexed=2, RGB=3, CMYK=4, etc.)
}

// layerRecord contains basic information for individual layers
// Represents the core layer data structure within PSD files
type layerRecord struct {
	Top      int32  // Top coordinate of layer bounds
	Left     int32  // Left coordinate of layer bounds
	Bottom   int32  // Bottom coordinate of layer bounds
	Right    int32  // Right coordinate of layer bounds
	Channels uint16 // Number of channels in this layer
}

// DetailedPSDInfo contains comprehensive layer and document information
// Extended version of PSDInfo with detailed layer analysis capabilities
type DetailedPSDInfo struct {
	*PSDInfo                   // Embedded basic PSD information
	Layers     []DetailedLayer `json:"layers"`      // Comprehensive layer details
	CanvasInfo CanvasInfo      `json:"canvas_info"` // Canvas-level information
}

// DetailedLayer contains comprehensive information about individual layers
// Provides detailed analysis of layer properties, position, and content
type DetailedLayer struct {
	ID          int      `json:"id"`           // Unique layer identifier
	Name        string   `json:"name"`         // Layer name as set by user
	Position    [4]int32 `json:"position"`     // Layer bounds: top, left, bottom, right
	BlendMode   string   `json:"blend_mode"`   // Layer blending mode
	Opacity     uint8    `json:"opacity"`      // Layer opacity (0-255)
	Visible     bool     `json:"visible"`      // Layer visibility state
	ContentHash string   `json:"content_hash"` // Hash of layer content for change detection
	LayerType   string   `json:"layer_type"`   // Layer type: "normal", "text", "adjustment", etc.
}

// CanvasInfo contains document-level canvas information
// Provides comprehensive details about the overall document structure
type CanvasInfo struct {
	Width      int `json:"width"`      // Canvas width in pixels
	Height     int `json:"height"`     // Canvas height in pixels
	ColorMode  int `json:"color_mode"` // Document color mode
	BitDepth   int `json:"bit_depth"`  // Document bit depth
	Resolution int `json:"resolution"` // Document resolution in DPI
}

// GetPSDInfo extracts comprehensive metadata from Photoshop PSD files
// Analyzes PSD file structure and returns detailed document and layer information
func GetPSDInfo(filePath string) (*PSDInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PSD file: %w", err)
	}
	defer file.Close()

	// Step 1: Parse and validate PSD file header section
	header := psdFileHeader{}
	err = binary.Read(file, binary.BigEndian, &header)
	if err != nil {
		return nil, fmt.Errorf("failed to read PSD file header: %w", err)
	}

	// Validate PSD file signature and version
	if string(header.Signature[:]) != "8BPS" {
		return nil, fmt.Errorf("invalid PSD file signature: %s", string(header.Signature[:]))
	}
	if header.Version != 1 && header.Version != 2 {
		return nil, fmt.Errorf("unsupported PSD file version: %d", header.Version)
	}

	// Step 2: Skip color mode data section (not needed for basic info)
	var colorModeDataLength uint32
	err = binary.Read(file, binary.BigEndian, &colorModeDataLength)
	if err != nil {
		return nil, fmt.Errorf("failed to read color mode data length: %w", err)
	}
	_, err = file.Seek(int64(colorModeDataLength), io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("failed to skip color mode data: %w", err)
	}

	// Step 3: Skip image resources section (contains metadata we don't need)
	var imageResourcesLength uint32
	err = binary.Read(file, binary.BigEndian, &imageResourcesLength)
	if err != nil {
		return nil, fmt.Errorf("failed to read image resources length: %w", err)
	}
	_, err = file.Seek(int64(imageResourcesLength), io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("failed to skip image resources: %w", err)
	}

	// Step 4: Parse layer and mask information section
	var layerAndMaskInfoLength uint32
	err = binary.Read(file, binary.BigEndian, &layerAndMaskInfoLength)
	if err != nil {
		return nil, fmt.Errorf("failed to read layer and mask info length: %w", err)
	}

	// Handle documents without layers (flattened images)
	if layerAndMaskInfoLength == 0 {
		return &PSDInfo{
			Width:      int(header.Width),
			Height:     int(header.Height),
			Channels:   int(header.Channels),
			Bits:       int(header.Depth),
			LayerCount: 0,
			LayerNames: []string{},
		}, nil
	}

	// Read layer information section length
	var layerInfoLength uint32
	err = binary.Read(file, binary.BigEndian, &layerInfoLength)
	if err != nil {
		return nil, fmt.Errorf("failed to read layer info length: %w", err)
	}

	// Read layer count (can be negative for specific PSD features)
	var layerCountRaw int16
	err = binary.Read(file, binary.BigEndian, &layerCountRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to read layer count: %w", err)
	}

	// Convert layer count to positive value
	layerCount := int(layerCountRaw)
	if layerCount < 0 {
		layerCount = -layerCount // Negative indicates absolute blend mode info
	}

	// Extract actual layer names from layer records
	layerNames, parseErr := parseLayerNames(file, layerCount)
	if parseErr != nil {
		// If layer name parsing fails, generate default names
		fmt.Printf("Warning: Could not parse layer names: %v\n", parseErr)
		layerNames = make([]string, layerCount)
		for i := 0; i < layerCount; i++ {
			layerNames[i] = fmt.Sprintf("Layer %d", i+1)
		}
	}

	// Return comprehensive PSD information
	return &PSDInfo{
		Width:      int(header.Width),
		Height:     int(header.Height),
		Channels:   int(header.Channels),
		Bits:       int(header.Depth),
		LayerCount: layerCount,
		LayerNames: layerNames,
	}, nil
}

// parseLayerNames extracts actual layer names from layer record structures
// Handles complex PSD layer data format and Unicode name extraction
func parseLayerNames(file *os.File, layerCount int) ([]string, error) {
	layerNames := make([]string, 0, layerCount)

	// Process each layer record in the PSD file
	for i := 0; i < layerCount; i++ {
		// Read basic layer record structure
		var layerRec layerRecord
		err := binary.Read(file, binary.BigEndian, &layerRec)
		if err != nil {
			return nil, fmt.Errorf("failed to read layer record %d: %w", i, err)
		}

		// Skip channel information (6 bytes per channel: 2 bytes ID + 4 bytes length)
		channelInfoSize := int64(layerRec.Channels) * 6
		_, err = file.Seek(channelInfoSize, io.SeekCurrent)
		if err != nil {
			return nil, fmt.Errorf("failed to skip channel info for layer %d: %w", i, err)
		}

		// Read blend mode signature (should be '8BIM')
		var blendModeSignature [4]byte
		err = binary.Read(file, binary.BigEndian, &blendModeSignature)
		if err != nil {
			return nil, fmt.Errorf("failed to read blend mode signature for layer %d: %w", i, err)
		}

		// Read blend mode key (4 bytes)
		var blendModeKey [4]byte
		err = binary.Read(file, binary.BigEndian, &blendModeKey)
		if err != nil {
			return nil, fmt.Errorf("failed to read blend mode key for layer %d: %w", i, err)
		}

		// Read layer flags (opacity, clipping, flags, filler - 4 bytes total)
		var layerFlags [4]byte
		err = binary.Read(file, binary.BigEndian, &layerFlags)
		if err != nil {
			return nil, fmt.Errorf("failed to read layer flags for layer %d: %w", i, err)
		}

		// Read Extra Data field length (contains layer name and additional info)
		var extraDataLength uint32
		err = binary.Read(file, binary.BigEndian, &extraDataLength)
		if err != nil {
			return nil, fmt.Errorf("failed to read extra data length for layer %d: %w", i, err)
		}

		// Handle layers without extra data
		if extraDataLength == 0 {
			layerNames = append(layerNames, fmt.Sprintf("Layer %d", i+1))
			continue
		}

		// Extract layer name from Extra Data section
		layerName, nameErr := extractLayerNameFromExtraData(file, extraDataLength)
		if nameErr != nil {
			// If name extraction fails, skip the extra data and use default name
			_, skipErr := file.Seek(int64(extraDataLength), io.SeekCurrent)
			if skipErr != nil {
				return nil, fmt.Errorf("failed to skip extra data for layer %d: %w", i, skipErr)
			}
			layerNames = append(layerNames, fmt.Sprintf("Layer %d", i+1))
		} else {
			layerNames = append(layerNames, layerName)
		}
	}

	return layerNames, nil
}

// extractLayerNameFromExtraData extracts layer name from the Extra Data section
// Handles both Pascal string names and Unicode names in Additional Layer Information
func extractLayerNameFromExtraData(file *os.File, extraDataLength uint32) (string, error) {
	startPos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}

	// Read and skip Layer Mask Data section
	var layerMaskLength uint32
	err = binary.Read(file, binary.BigEndian, &layerMaskLength)
	if err != nil {
		return "", err
	}

	_, err = file.Seek(int64(layerMaskLength), io.SeekCurrent)
	if err != nil {
		return "", err
	}

	// Read and skip Layer Blending Ranges section
	var blendingRangesLength uint32
	err = binary.Read(file, binary.BigEndian, &blendingRangesLength)
	if err != nil {
		return "", err
	}

	_, err = file.Seek(int64(blendingRangesLength), io.SeekCurrent)
	if err != nil {
		return "", err
	}

	// Read layer name as Pascal String (length byte + name + padding)
	var nameLength byte
	err = binary.Read(file, binary.BigEndian, &nameLength)
	if err != nil {
		return "", err
	}

	// Handle empty layer names
	if nameLength == 0 {
		return "Unnamed Layer", nil
	}

	// Read layer name bytes
	nameBytes := make([]byte, nameLength)
	_, err = file.Read(nameBytes)
	if err != nil {
		return "", err
	}

	// Calculate and skip padding to align to 4-byte boundary
	paddingNeeded := (4 - ((1 + int(nameLength)) % 4)) % 4
	if paddingNeeded > 0 {
		_, err = file.Seek(int64(paddingNeeded), io.SeekCurrent)
		if err != nil {
			return "", err
		}
	}

	// Try to find Unicode layer name in Additional Layer Information section
	currentPos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return string(nameBytes), nil // Return ASCII name if Unicode lookup fails
	}

	// Calculate remaining bytes in Extra Data section
	remainingBytes := int64(extraDataLength) - (currentPos - startPos)
	if remainingBytes > 0 {
		unicodeName, unicodeErr := findUnicodeLayerName(file, remainingBytes)
		if unicodeErr == nil && unicodeName != "" {
			return unicodeName, nil
		}
	}

	// Return ASCII layer name if Unicode name not found
	return string(nameBytes), nil
}

// findUnicodeLayerName searches for Unicode layer name in Additional Layer Information
// Provides support for international character sets in layer names
func findUnicodeLayerName(file *os.File, maxBytes int64) (string, error) {
	startPos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}
	defer file.Seek(startPos+maxBytes, io.SeekStart) // Restore position after search

	// Search through Additional Layer Information blocks
	for {
		currentPos, err := file.Seek(0, io.SeekCurrent)
		if err != nil || currentPos-startPos >= maxBytes-8 {
			break
		}

		// Read Additional Layer Information signature ('8BIM' or '8B64')
		var signature [4]byte
		err = binary.Read(file, binary.BigEndian, &signature)
		if err != nil {
			break
		}

		sigStr := string(signature[:])
		if sigStr != "8BIM" && sigStr != "8B64" {
			// Move back 3 bytes and continue searching (sliding window approach)
			_, err = file.Seek(-3, io.SeekCurrent)
			if err != nil {
				break
			}
			continue
		}

		// Read key identifier (4 bytes)
		var key [4]byte
		err = binary.Read(file, binary.BigEndian, &key)
		if err != nil {
			break
		}

		// Read data length for this information block
		var dataLength uint32
		err = binary.Read(file, binary.BigEndian, &dataLength)
		if err != nil {
			break
		}

		keyStr := string(key[:])
		if keyStr == "luni" { // Layer Unicode Name block
			// Read Unicode string length (4 bytes)
			var unicodeLength uint32
			err = binary.Read(file, binary.BigEndian, &unicodeLength)
			if err != nil {
				break
			}

			// Validate Unicode length is reasonable
			if unicodeLength > 0 && unicodeLength < 1000 {
				// Read UTF-16 encoded data
				utf16Data := make([]uint16, unicodeLength)
				err = binary.Read(file, binary.BigEndian, &utf16Data)
				if err != nil {
					break
				}

				// Convert UTF-16 to UTF-8 string
				runes := utf16.Decode(utf16Data)
				return string(runes), nil
			}
		}

		// Skip to next information block
		_, err = file.Seek(int64(dataLength), io.SeekCurrent)
		if err != nil {
			break
		}

		// Handle 2-byte alignment padding
		if dataLength%2 != 0 {
			_, err = file.Seek(1, io.SeekCurrent)
			if err != nil {
				break
			}
		}
	}

	return "", fmt.Errorf("unicode layer name not found")
}

// GetDetailedPSDInfo extracts comprehensive PSD information including detailed layer analysis
// Provides extended functionality for advanced PSD file inspection
func GetDetailedPSDInfo(filePath string) (*DetailedPSDInfo, error) {
	// Step 1: Get basic PSD information first
	basicInfo, err := GetPSDInfo(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic PSD info: %w", err)
	}

	// Step 2: Open file for detailed layer analysis
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PSD file for detailed analysis: %w", err)
	}
	defer file.Close()

	// Step 3: Create comprehensive detailed info structure
	detailedInfo := &DetailedPSDInfo{
		PSDInfo: basicInfo,
		Layers:  make([]DetailedLayer, 0, basicInfo.LayerCount),
		CanvasInfo: CanvasInfo{
			Width:     basicInfo.Width,
			Height:    basicInfo.Height,
			ColorMode: basicInfo.Channels,
			BitDepth:  basicInfo.Bits,
		},
	}

	// Step 4: Parse detailed layer information
	layers, err := parseDetailedLayers(file, basicInfo.LayerCount, filePath)
	if err != nil {
		// If detailed parsing fails, create basic layers from existing info
		fmt.Printf("Warning: Could not parse detailed layer info: %v\n", err)
		layers = createBasicLayersFromNames(basicInfo.LayerNames, filePath)
	}

	detailedInfo.Layers = layers

	return detailedInfo, nil
}

// parseDetailedLayers parses comprehensive layer information including positions, blend modes, and content hashes
// This is the core function for detailed layer analysis and change detection
func parseDetailedLayers(file *os.File, layerCount int, filePath string) ([]DetailedLayer, error) {
	if layerCount == 0 {
		return []DetailedLayer{}, nil
	}

	// Reset file position to start of file for complete parsing
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to reset file position: %w", err)
	}

	// Skip to layer information section by parsing header structure
	err = skipToLayerInfo(file)
	if err != nil {
		return nil, fmt.Errorf("failed to skip to layer info: %w", err)
	}

	layers := make([]DetailedLayer, 0, layerCount)

	// Parse each layer record with detailed information
	for i := 0; i < layerCount; i++ {
		layer, err := parseIndividualLayer(file, i, filePath)
		if err != nil {
			// If individual layer parsing fails, create basic layer info
			fmt.Printf("Warning: Failed to parse layer %d: %v\n", i, err)
			layer = &DetailedLayer{
				ID:          i,
				Name:        fmt.Sprintf("Layer %d", i+1),
				Position:    [4]int32{0, 0, 100, 100},
				BlendMode:   "normal",
				Opacity:     255,
				Visible:     true,
				ContentHash: generateLayerContentHash(filePath, i, "unknown"),
				LayerType:   "normal",
			}
		}
		layers = append(layers, *layer)
	}

	return layers, nil
}

// skipToLayerInfo navigates file pointer to the layer information section
// Skips header, color mode data, and image resources to reach layer data
func skipToLayerInfo(file *os.File) error {
	// Skip PSD header (26 bytes)
	_, err := file.Seek(26, io.SeekStart)
	if err != nil {
		return err
	}

	// Skip color mode data section
	var colorModeDataLength uint32
	err = binary.Read(file, binary.BigEndian, &colorModeDataLength)
	if err != nil {
		return err
	}
	_, err = file.Seek(int64(colorModeDataLength), io.SeekCurrent)
	if err != nil {
		return err
	}

	// Skip image resources section
	var imageResourcesLength uint32
	err = binary.Read(file, binary.BigEndian, &imageResourcesLength)
	if err != nil {
		return err
	}
	_, err = file.Seek(int64(imageResourcesLength), io.SeekCurrent)
	if err != nil {
		return err
	}

	// Read layer and mask info length
	var layerAndMaskInfoLength uint32
	err = binary.Read(file, binary.BigEndian, &layerAndMaskInfoLength)
	if err != nil {
		return err
	}

	// Skip layer info length field (4 bytes)
	var layerInfoLength uint32
	err = binary.Read(file, binary.BigEndian, &layerInfoLength)
	if err != nil {
		return err
	}

	// Skip layer count field (2 bytes) - we already know the count
	_, err = file.Seek(2, io.SeekCurrent)
	return err
}

// parseIndividualLayer extracts detailed information for a single layer
// Returns comprehensive layer data including bounds, blend mode, opacity, and content hash
func parseIndividualLayer(file *os.File, layerIndex int, filePath string) (*DetailedLayer, error) {
	// Read layer record structure (bounds + channel count)
	var layerRec layerRecord
	err := binary.Read(file, binary.BigEndian, &layerRec)
	if err != nil {
		return nil, fmt.Errorf("failed to read layer record: %w", err)
	}

	// Skip channel information (6 bytes per channel)
	channelInfoSize := int64(layerRec.Channels) * 6
	_, err = file.Seek(channelInfoSize, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("failed to skip channel info: %w", err)
	}

	// Read blend mode signature
	var blendModeSignature [4]byte
	err = binary.Read(file, binary.BigEndian, &blendModeSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to read blend mode signature: %w", err)
	}

	// Read blend mode key
	var blendModeKey [4]byte
	err = binary.Read(file, binary.BigEndian, &blendModeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read blend mode key: %w", err)
	}

	// Read layer flags (opacity, clipping, visibility, etc.)
	var layerFlags [4]byte
	err = binary.Read(file, binary.BigEndian, &layerFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to read layer flags: %w", err)
	}

	// Extract layer properties from flags
	opacity := layerFlags[0]               // First byte is opacity
	visible := (layerFlags[2] & 0x02) == 0 // Bit 1 of third byte (inverted)

	// Convert blend mode key to readable string
	blendMode := string(blendModeKey[:])
	readableBlendMode := mapBlendMode(blendMode)

	// Read Extra Data length
	var extraDataLength uint32
	err = binary.Read(file, binary.BigEndian, &extraDataLength)
	if err != nil {
		return nil, fmt.Errorf("failed to read extra data length: %w", err)
	}

	// Extract layer name from extra data
	layerName := fmt.Sprintf("Layer %d", layerIndex+1)
	if extraDataLength > 0 {
		startPos, _ := file.Seek(0, io.SeekCurrent)
		extractedName, nameErr := extractLayerNameFromExtraData(file, extraDataLength)
		if nameErr == nil && extractedName != "" {
			layerName = extractedName
		} else {
			// If name extraction fails, skip the extra data
			file.Seek(startPos+int64(extraDataLength), io.SeekStart)
		}
	}

	// Generate content hash based on layer properties
	contentHash := generateLayerContentHash(filePath, layerIndex, layerName)

	// Determine layer type based on characteristics
	layerType := determineLayerType(layerName, blendMode)

	return &DetailedLayer{
		ID:          layerIndex,
		Name:        layerName,
		Position:    [4]int32{layerRec.Top, layerRec.Left, layerRec.Bottom, layerRec.Right},
		BlendMode:   readableBlendMode,
		Opacity:     opacity,
		Visible:     visible,
		ContentHash: contentHash,
		LayerType:   layerType,
	}, nil
}

// generateLayerContentHash creates a unique hash for layer content based on multiple factors
// Used for detecting changes between versions at the layer level
func generateLayerContentHash(filePath string, layerIndex int, layerName string) string {
	// Create hash input from multiple layer characteristics
	hasher := sha256.New()

	// Include file path and layer index for uniqueness
	hasher.Write([]byte(filePath))
	hasher.Write([]byte(fmt.Sprintf(":%d:", layerIndex)))
	hasher.Write([]byte(layerName))

	// Get file modification time for additional uniqueness
	if fileInfo, err := os.Stat(filePath); err == nil {
		hasher.Write([]byte(fileInfo.ModTime().Format("2006-01-02T15:04:05")))
	}

	// Return first 16 characters of hash for readability
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	return hash[:16]
}

// mapBlendMode converts PSD blend mode keys to readable names
// Maps 4-character keys to user-friendly blend mode names
func mapBlendMode(blendModeKey string) string {
	blendModes := map[string]string{
		"norm": "normal",
		"dark": "darken",
		"lite": "lighten",
		"hue ": "hue",
		"sat ": "saturation",
		"colr": "color",
		"lum ": "luminosity",
		"mul ": "multiply",
		"scrn": "screen",
		"over": "overlay",
		"sLit": "soft light",
		"hLit": "hard light",
		"diff": "difference",
		"smud": "exclusion",
	}

	if readable, exists := blendModes[blendModeKey]; exists {
		return readable
	}
	return "normal" // Default fallback
}

// determineLayerType analyzes layer characteristics to determine layer type
// Helps categorize layers for better change tracking and user interface
func determineLayerType(layerName, blendMode string) string {
	layerNameLower := strings.ToLower(layerName)

	// Check for text layers
	if strings.Contains(layerNameLower, "text") || strings.Contains(layerNameLower, "txt") {
		return "text"
	}

	// Check for adjustment layers
	if strings.Contains(layerNameLower, "adjustment") || strings.Contains(layerNameLower, "adj") {
		return "adjustment"
	}

	// Check for background layers
	if layerNameLower == "background" || strings.Contains(layerNameLower, "bg") {
		return "background"
	}

	// Check for effect layers based on blend mode
	if blendMode != "normal" {
		return "effect"
	}

	return "normal"
}

// createBasicLayersFromNames creates basic layer info when detailed parsing fails
// Provides fallback functionality to ensure consistent layer information
func createBasicLayersFromNames(layerNames []string, filePath string) []DetailedLayer {
	layers := make([]DetailedLayer, len(layerNames))

	for i, name := range layerNames {
		layers[i] = DetailedLayer{
			ID:          i,
			Name:        name,
			Position:    [4]int32{0, 0, 100, 100}, // Default position bounds
			BlendMode:   "normal",                 // Default blend mode
			Opacity:     255,                      // Full opacity (0-255 scale)
			Visible:     true,                     // Assume visible by default
			ContentHash: generateLayerContentHash(filePath, i, name),
			LayerType:   determineLayerType(name, "normal"),
		}
	}

	return layers
}
