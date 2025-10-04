package cmd

import (
	"fmt"
	"os"
	"strings"

	"dgit/internal/staging"
	"github.com/spf13/cobra"
)

// AddCmd adds design files to the staging area for the next commit
var AddCmd = &cobra.Command{
	Use:   "add [files...]",
	Short: "Add design files to the staging area",
	Long: `Add design files to the staging area for the next commit.

Examples:
  dgit add logo.ai     # Add specific file
  dgit add .           # Add all design files
  dgit add *.psd       # Add all PSD files`,
	Args: cobra.MinimumNArgs(1),
	Run:  runAdd,
}

// runAdd stages files for the next commit
func runAdd(cmd *cobra.Command, args []string) {
	if !isInDgitRepository() {
		printError("not a dgit repository (or any of the parent directories)")
		printSuggestion("Run 'dgit init' to initialize a repository")
		os.Exit(1)
	}

	dgitDir := findDgitDirectory()
	stagingArea := staging.NewStagingArea(dgitDir)

	if err := stagingArea.LoadStaging(); err != nil {
		printError(fmt.Sprintf("loading staging area: %v", err))
		os.Exit(1)
	}

	var allAddedFiles []string
	var allFailedFiles = make(map[string]error)

	for _, arg := range args {
		result, err := stagingArea.AddPattern(arg)
		if err != nil {
			printError(fmt.Sprintf("adding '%s': %v", arg, err))
			continue
		}

		allAddedFiles = append(allAddedFiles, result.AddedFiles...)

		for file, fileErr := range result.FailedFiles {
			printWarning(fmt.Sprintf("failed to add %s: %v", file, fileErr))
			allFailedFiles[file] = fileErr
		}
	}

	if err := stagingArea.SaveStaging(); err != nil {
		printError(fmt.Sprintf("saving staging area: %v", err))
		os.Exit(1)
	}

	if len(allAddedFiles) > 0 {
		printSuccess(fmt.Sprintf("Added %d file(s) to staging area:", len(allAddedFiles)))
		for _, file := range allAddedFiles {
			fmt.Printf("  + %s\n", file)
		}
		fmt.Println()
		printStagingStatus(stagingArea)
	} else {
		fmt.Println("No files were added to staging area.")
	}
}

// printStagingStatus displays staged files with metadata
func printStagingStatus(stagingArea *staging.StagingArea) {
	stagedFiles := stagingArea.GetStagedFiles()
	if len(stagedFiles) == 0 {
		fmt.Println("No files staged for commit.")
		return
	}

	fmt.Printf("Files staged for commit (%d):\n", len(stagedFiles))
	for _, file := range stagedFiles {
		fmt.Printf("  %s (%s, %.2f KB)\n",
			file.Path,
			strings.ToUpper(file.FileType),
			float64(file.Size)/1024)
	}
}
