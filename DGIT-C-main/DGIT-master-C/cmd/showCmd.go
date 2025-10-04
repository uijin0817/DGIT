package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"dgit/internal/log"
	"dgit/internal/scanner"

	"github.com/spf13/cobra"
)

// ShowCmd represents the show command for detailed information display
var ShowCmd = &cobra.Command{
	Use:   "show <target>",
	Short: "Show detailed information",
	Long: `Show detailed information about files or commits.

Examples:
  dgit show design.psd        # Detailed file analysis
  dgit show dfb6ae0          # Commit information
  dgit show v1               # Version information
  dgit show --name-only v1   # List files in commit`,
	Args: cobra.ExactArgs(1),
	Run:  runShow,
}

func init() {
	ShowCmd.Flags().Bool("name-only", false, "Show only file names (for commits)")
	ShowCmd.Flags().Bool("layers", false, "Show layer information (for design files)")
	ShowCmd.Flags().Bool("json", false, "Output in JSON format") // 추가 필요
}

// runShow executes the show command for files or commits
func runShow(cmd *cobra.Command, args []string) {
	target := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if isFilePath(target) {
		showFileDetails(target, cmd)
	} else {
		showCommitDetails(target, cmd, jsonOutput) // 파라미터 추가
	}
}

// showFileDetails displays comprehensive file analysis
func showFileDetails(filePath string, cmd *cobra.Command) {
	if !fileExists(filePath) {
		printError(fmt.Sprintf("file not found: %s", filePath))
		os.Exit(1)
	}

	if !scanner.IsDesignFile(filePath) {
		printError(fmt.Sprintf("not a design file: %s", filePath))
		os.Exit(1)
	}

	fmt.Printf("Analyzing file: %s\n\n", filePath)

	// Use detailed scanner for comprehensive analysis
	detailedScanner := scanner.NewDetailedScanner()
	fileInfo, err := detailedScanner.AnalyzeFile(filePath)
	if err != nil {
		printError(fmt.Sprintf("analysis failed: %v", err))
		os.Exit(1)
	}

	printFileDetails(fileInfo, cmd)
}

// showCommitDetails displays commit information
func showCommitDetails(commitRef string, cmd *cobra.Command, jsonOutput bool) {
	dgitDir := checkDgitRepository()
	logManager := log.NewLogManager(dgitDir)

	commit, err := findCommit(logManager, commitRef)
	if err != nil {
		printError(fmt.Sprintf("commit '%s' not found", commitRef))
		os.Exit(1)
	}

	nameOnly, _ := cmd.Flags().GetBool("name-only")
	if nameOnly {
		printCommitFileNames(commit, jsonOutput) // 파라미터 추가
	} else {
		printCommitDetails(commit, jsonOutput) // 전체 정보도 JSON 지원
	}
}

// printFileDetails displays detailed file information
func printFileDetails(fileInfo *scanner.DetailedFileInfo, cmd *cobra.Command) {
	// Basic file information
	fmt.Printf("File: %s\n", fileInfo.Path)
	fmt.Printf("Type: %s\n", getFileTypeDescription(fileInfo.Type))
	fmt.Printf("Size: %.1f MB\n", float64(fileInfo.FileSize)/(1024*1024))

	if fileInfo.Dimensions != "Unknown" {
		fmt.Printf("Dimensions: %s\n", fileInfo.Dimensions)
	}
	if fileInfo.ColorMode != "Unknown" {
		fmt.Printf("Color Mode: %s\n", fileInfo.ColorMode)
	}
	if fileInfo.Version != "Unknown" {
		fmt.Printf("Application: %s\n", fileInfo.Version)
	}

	// Layer information
	if fileInfo.Layers > 0 {
		fmt.Printf("\nLayers: %d\n", fileInfo.Layers)

		showLayers, _ := cmd.Flags().GetBool("layers")
		if showLayers && len(fileInfo.LayerNames) > 0 {
			fmt.Println("Layer structure:")
			for i, layerName := range fileInfo.LayerNames {
				fmt.Printf("  %d. %s\n", i+1, layerName)
			}
		}
	}

	if fileInfo.Artboards > 1 {
		fmt.Printf("Artboards: %d\n", fileInfo.Artboards)
	}

	fmt.Printf("\nAnalysis completed\n")
}

// printCommitDetails displays comprehensive commit information
func printCommitDetails(commit *log.Commit, jsonOutput bool) {
	if jsonOutput {
		// JSON 출력
		result := map[string]interface{}{
			"commit":      commit.Hash,
			"version":     commit.Version,
			"author":      commit.Author,
			"date":        commit.Timestamp.Format("Mon Jan 2 15:04:05 2006"),
			"message":     commit.Message,
			"files_count": commit.FilesCount,
			"files":       commit.Metadata,
		}

		if commit.CompressionInfo != nil {
			compressionPercent := (1.0 - commit.CompressionInfo.CompressionRatio) * 100
			result["compression"] = map[string]interface{}{
				"strategy":     commit.CompressionInfo.Strategy,
				"saved":        fmt.Sprintf("%.1f%%", compressionPercent),
				"base_version": commit.CompressionInfo.BaseVersion,
			}
		}

		if jsonData, err := json.Marshal(result); err == nil {
			fmt.Println(string(jsonData))
		}
		return
	}

	// 기존 텍스트 출력
	fmt.Printf("commit %s (v%d)\n", commit.Hash, commit.Version)
	fmt.Printf("Author: %s\n", commit.Author)
	fmt.Printf("Date: %s\n", commit.Timestamp.Format("Mon Jan 2 15:04:05 2006"))
	fmt.Printf("\n    %s\n\n", commit.Message)

	// Storage information
	if commit.CompressionInfo != nil {
		compressionPercent := (1.0 - commit.CompressionInfo.CompressionRatio) * 100
		fmt.Printf("Storage: %s compression (%.1f%% saved)\n",
			commit.CompressionInfo.Strategy, compressionPercent)
		if commit.CompressionInfo.BaseVersion > 0 {
			fmt.Printf("Base version: v%d\n", commit.CompressionInfo.BaseVersion)
		}
		fmt.Println()
	}

	// Files in commit
	fmt.Printf("Files (%d):\n", commit.FilesCount)
	for fileName, metadata := range commit.Metadata {
		fmt.Printf("  %s", fileName)
		if metaMap, ok := metadata.(map[string]interface{}); ok {
			printStoredMetadata(metaMap) // 기존 함수 활용
		}
		fmt.Println()
	}
}

// printCommitFileNames displays only file names from commit
func printCommitFileNames(commit *log.Commit, jsonOutput bool) {
	if jsonOutput {
		fileNames := make([]string, 0, len(commit.Metadata))
		for fileName := range commit.Metadata {
			fileNames = append(fileNames, fileName)
		}

		result := map[string]interface{}{
			"files":   fileNames,
			"commit":  commit.Hash,
			"version": commit.Version,
		}

		if jsonData, err := json.Marshal(result); err == nil {
			fmt.Println(string(jsonData))
		}
		return
	}

	// 기존 텍스트 출력 유지
	for fileName := range commit.Metadata {
		fmt.Println(fileName)
	}
}

// Helper functions
func isFilePath(target string) bool {
	return strings.Contains(target, ".") || strings.Contains(target, "/")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func findCommit(logManager *log.LogManager, commitRef string) (*log.Commit, error) {
	// Try by hash first
	commit, err := logManager.GetCommitByHash(commitRef)
	if err == nil {
		return commit, nil
	}

	// Try by version number
	if version, parseErr := parseVersion(commitRef); parseErr == nil {
		return logManager.GetCommit(version)
	}

	return nil, fmt.Errorf("commit not found")
}

func parseVersion(versionStr string) (int, error) {
	if strings.HasPrefix(versionStr, "v") {
		versionStr = strings.TrimPrefix(versionStr, "v")
	}
	return strconv.Atoi(versionStr)
}

func getFileTypeDescription(fileType string) string {
	descriptions := map[string]string{
		"psd":      "Adobe Photoshop Document",
		"ai":       "Adobe Illustrator File",
		"sketch":   "Sketch Design File",
		"fig":      "Figma Design File",
		"xd":       "Adobe XD Document",
		"afdesign": "Affinity Designer File",
		"afphoto":  "Affinity Photo File",
	}

	if desc, exists := descriptions[fileType]; exists {
		return desc
	}
	return strings.ToUpper(fileType) + " File"
}

func printStoredMetadata(metaMap map[string]interface{}) {
	var details []string

	if dimensions, ok := metaMap["dimensions"].(string); ok && dimensions != "Unknown" {
		details = append(details, dimensions)
	}
	if layers, ok := metaMap["layers"].(float64); ok && layers > 0 {
		details = append(details, fmt.Sprintf("%.0f layers", layers))
	}
	if colorMode, ok := metaMap["color_mode"].(string); ok && colorMode != "Unknown" {
		details = append(details, colorMode)
	}

	if len(details) > 0 {
		fmt.Printf(" (%s)", strings.Join(details, ", "))
	}
}
