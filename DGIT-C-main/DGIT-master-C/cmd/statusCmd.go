package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dgit/internal/log"
	"dgit/internal/scanner"
	"dgit/internal/staging"
	"dgit/internal/status"

	"github.com/spf13/cobra"
)

// StatusCmd shows the working tree status
var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the working tree status",
	Long: `Display the current status of the repository including:
- Files staged for commit
- Modified files not yet staged  
- Untracked design files
- Deleted files

Shows metadata changes for design files such as layer count, 
dimension changes, and color mode changes.`,
	Run: runStatus,
}

// runStatus shows repository status with design file metadata changes
func runStatus(cmd *cobra.Command, args []string) {
	dgitDir := checkDgitRepository()

	stagingArea := staging.NewStagingArea(dgitDir)
	statusManager := status.NewStatusManager(dgitDir)
	logManager := log.NewLogManager(dgitDir)

	if err := stagingArea.LoadStaging(); err != nil {
		printError(fmt.Sprintf("loading staging area: %v", err))
		os.Exit(1)
	}

	currentVersion := logManager.GetCurrentVersion()
	fmt.Printf("On version %d\n\n", currentVersion+1)

	if !stagingArea.IsEmpty() {
		fmt.Println("Changes to be committed:")
		printStatusStagingInfo(stagingArea)
		fmt.Println()
	} else {
		fmt.Println("No changes staged for commit.")
		fmt.Println()
	}

	currentWorkDir, _ := os.Getwd()
	currentDirFiles := scanCurrentDirectory(currentWorkDir)

	result, err := statusManager.CompareWithCommit(currentVersion, currentDirFiles)
	if err != nil {
		printWarning(fmt.Sprintf("Failed to compare with last commit: %v", err))
		return
	}

	var lastCommit *log.Commit
	if currentVersion > 0 {
		lastCommit, err = logManager.GetCommit(currentVersion)
		if err != nil {
			printWarning(fmt.Sprintf("Failed to load last commit for metadata comparison: %v", err))
		}
	}

	result.ModifiedFiles = filterStagedFiles(result.ModifiedFiles, stagingArea)
	result.UntrackedFiles = filterStagedFiles(result.UntrackedFiles, stagingArea)
	result.DeletedFiles = filterStagedFiles(result.DeletedFiles, stagingArea)

	if len(result.ModifiedFiles) > 0 {
		fmt.Println("Changes not staged for commit:")
		for _, fileStatus := range result.ModifiedFiles {
			metadataSummary := getMetadataChangeSummary(fileStatus.Path, lastCommit, currentWorkDir)
			fmt.Printf("  modified: %s%s\n", fileStatus.Path, metadataSummary)
		}
		fmt.Println()
	} else {
		fmt.Println("No changes not staged for commit.")
	}

	if len(result.UntrackedFiles) > 0 {
		fmt.Println("Untracked files:")
		for _, fileStatus := range result.UntrackedFiles {
			fileType := getStatusFileType(fileStatus.Path)
			fmt.Printf("  [%s] %s\n", fileType, fileStatus.Path)
		}
		fmt.Println()
	} else {
		fmt.Println("No untracked files.")
	}

	if len(result.DeletedFiles) > 0 {
		fmt.Println("Deleted files:")
		for _, fileStatus := range result.DeletedFiles {
			fmt.Printf("  deleted: %s\n", fileStatus.Path)
		}
		fmt.Println()
	} else {
		fmt.Println("No deleted files.")
	}

	fmt.Println("Commands:")
	fmt.Println("   Use 'dgit add <file>' to stage files for commit")
	fmt.Println("   Use 'dgit commit' to commit staged changes")
	if len(result.ModifiedFiles) > 0 || len(result.UntrackedFiles) > 0 {
		fmt.Println("   Use 'dgit scan' to analyze design file details")
	}
}

// scanCurrentDirectory scans for design files and returns their hashes
func scanCurrentDirectory(currentWorkDir string) map[string]string {
	currentDirFiles := make(map[string]string)

	filepath.Walk(currentWorkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".dgit" {
				return filepath.SkipDir
			}
			return nil
		}

		if scanner.IsDesignFile(path) {
			relPath, relErr := filepath.Rel(currentWorkDir, path)
			if relErr != nil {
				return nil
			}

			hash, hashErr := status.CalculateFileHash(path)
			if hashErr != nil {
				return nil
			}
			currentDirFiles[relPath] = hash
		}
		return nil
	})

	return currentDirFiles
}

// filterStagedFiles removes files that are already staged
func filterStagedFiles(files []status.FileStatus, stagingArea *staging.StagingArea) []status.FileStatus {
	var filtered []status.FileStatus
	for _, file := range files {
		if !stagingArea.HasFile(file.Path) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// getMetadataChangeSummary generates a summary of design file metadata changes
func getMetadataChangeSummary(filePath string, lastCommit *log.Commit, currentWorkDir string) string {
	if lastCommit == nil {
		return ""
	}

	currentFileInfo, err := scanner.NewFileScanner().ScanFile(filepath.Join(currentWorkDir, filePath))
	if err != nil {
		return ""
	}

	oldMetaRaw, ok := lastCommit.Metadata[filePath].(map[string]interface{})
	if !ok {
		return ""
	}

	oldLayers, _ := oldMetaRaw["layers"].(float64)
	oldArtboards, _ := oldMetaRaw["artboards"].(float64)
	oldDimensions, _ := oldMetaRaw["dimensions"].(string)
	oldColorMode, _ := oldMetaRaw["color_mode"].(string)

	var changes []string
	if oldLayers != float64(currentFileInfo.Layers) && currentFileInfo.Layers != 0 {
		changes = append(changes, fmt.Sprintf("Layers: %.0f→%d", oldLayers, currentFileInfo.Layers))
	}
	if oldArtboards != float64(currentFileInfo.Artboards) && currentFileInfo.Artboards != 0 {
		changes = append(changes, fmt.Sprintf("Artboards: %.0f→%d", oldArtboards, currentFileInfo.Artboards))
	}
	if oldDimensions != currentFileInfo.Dimensions && currentFileInfo.Dimensions != "Unknown" {
		changes = append(changes, fmt.Sprintf("Dimensions: %s→%s", oldDimensions, currentFileInfo.Dimensions))
	}
	if oldColorMode != currentFileInfo.ColorMode && currentFileInfo.ColorMode != "Unknown" {
		changes = append(changes, fmt.Sprintf("ColorMode: %s→%s", oldColorMode, currentFileInfo.ColorMode))
	}

	if len(changes) > 0 {
		return " (" + strings.Join(changes, ", ") + ")"
	}
	return ""
}

// getStatusFileType returns file type indicator for status display
func getStatusFileType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".ai":
		return "AI"
	case ".psd":
		return "PSD"
	case ".sketch":
		return "SKETCH"
	case ".fig":
		return "FIG"
	case ".xd":
		return "XD"
	default:
		return "FILE"
	}
}

// printStagingStatus displays files staged for commit
func printStatusStagingInfo(stagingArea *staging.StagingArea) {
	for _, file := range stagingArea.GetStagedFiles() {
		fileType := getStatusFileType(file.Path)
		fmt.Printf("  [%s] new file: %s\n", fileType, file.Path)
	}
}
