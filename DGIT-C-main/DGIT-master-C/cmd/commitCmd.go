package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	
	"dgit/internal/commit"
	"dgit/internal/staging"
	"github.com/spf13/cobra"
)

// CommitCmd represents the commit command for creating snapshots of design files
// Similar to 'git commit' but with design-specific features like metadata extraction
var CommitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Create a new commit with staged files",
	Long: `Create a new commit with all files currently in the staging area.

Examples:
  dgit commit "Logo design completed"
  dgit commit -m "Updated color scheme to brand guidelines"
  dgit commit                       # Opens editor for commit message

The commit will:
- Create a snapshot (ZIP) of all staged files
- Extract and store metadata for each design file  
- Generate a unique commit hash
- Clear the staging area`,
	Args: cobra.MaximumNArgs(1),  // Optional commit message as argument
	Run:  runCommit,
}

// init sets up command flags
func init() {
	// Add -m flag for commit message (similar to git)
	CommitCmd.Flags().StringP("message", "m", "", "Commit message")
}

// runCommit executes the commit command functionality
// Creates a snapshot of all staged files with metadata
func runCommit(cmd *cobra.Command, args []string) {
	// Ensure we're working within a DGit repository
	if !isInDgitRepository() {
		printError("not a dgit repository (or any of the parent directories)")
		printSuggestion("Run 'dgit init' to initialize a repository")
		os.Exit(1)
	}

	// Get repository and staging area
	dgitDir := findDgitDirectory()
	stagingArea := staging.NewStagingArea(dgitDir)
	
	// Load current staging area state
	if err := stagingArea.LoadStaging(); err != nil {
		printError(fmt.Sprintf("loading staging area: %v", err))
		os.Exit(1)
	}

	// Check if there are any files to commit
	if stagingArea.IsEmpty() {
		fmt.Println("No files staged for commit.")
		fmt.Println("   Use 'dgit add <files>' to stage files for commit.")
		os.Exit(1)
	}

	// Get commit message from various sources (args, flag, or interactive input)
	var message string
	if len(args) > 0 {
		// Message provided as argument
		message = args[0]
	} else if msgFlag, _ := cmd.Flags().GetString("message"); msgFlag != "" {
		// Message provided via -m flag
		message = msgFlag
	} else {
		// Interactive input for commit message
		fmt.Print("Enter commit message: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			printError(fmt.Sprintf("reading commit message: %v", err))
			os.Exit(1)
		}
		message = strings.TrimSpace(input)

		// Ensure message is not empty
		if message == "" {
			printError("commit message cannot be empty")
			os.Exit(1)
		}
	}

	// Get staged files for processing
	stagedFiles := stagingArea.GetStagedFiles()
	
	// Display DGit-style commit progress messages
	fmt.Printf("Creating commit with %d design files...\n", len(stagedFiles))
	fmt.Println("Analyzing design file metadata...")
	fmt.Println("Creating snapshot archive...")
	
	// Create the actual commit with metadata and snapshot
	commitManager := commit.NewCommitManager(dgitDir)
	newCommit, err := commitManager.CreateCommit(message, stagedFiles)
	if err != nil {
		printError(fmt.Sprintf("creating commit: %v", err))
		os.Exit(1)
	}

	// Clear staging area after successful commit
	if err := stagingArea.ClearStaging(); err != nil {
		printWarning(fmt.Sprintf("failed to clear staging area: %v", err))
	}

	// Display DGit-style success message with commit details
	fmt.Printf("\n")
	printGreen(fmt.Sprintf("Created commit %s", newCommit.Hash[:8]))
	fmt.Printf("%s\n", message)
	printCyan(fmt.Sprintf("Author: %s", newCommit.Author))
	
	// Show design-specific file details (unique to DGit!)
	printBlue(fmt.Sprintf("Design files (%d):", newCommit.FilesCount))
	for fileName, metadata := range newCommit.Metadata {
		if metaMap, ok := metadata.(map[string]interface{}); ok {
			// Get file type for display
			fileType := getFileType(fileName)
			
			// Extract metadata fields
			layers, _ := metaMap["layers"].(float64)
			dimensions, _ := metaMap["dimensions"].(string)
			colorMode, _ := metaMap["color_mode"].(string)
			
			fmt.Printf("   [%s] %s", fileType, fileName)
			
			// Build metadata details string
			var details []string
			if layers > 0 {
				details = append(details, fmt.Sprintf("%.0f layers", layers))
			}
			if dimensions != "Unknown" && dimensions != "" {
				details = append(details, dimensions)
			}
			if colorMode != "Unknown" && colorMode != "" {
				details = append(details, colorMode)
			}
			
			// Display metadata if available
			if len(details) > 0 {
				fmt.Printf(" (%s)", strings.Join(details, ", "))
			}
			fmt.Println()
		} else {
			// Fallback for files without metadata
			fmt.Printf("   %s\n", fileName)
		}
	}
	
	printGreen(fmt.Sprintf("Snapshot: %s", newCommit.SnapshotZip))
	printBold("Ready for collaboration!")
}

// getFileType returns file type string based on file extension
// Used for visual distinction of different design file types in commit output
func getFileType(fileName string) string {
	lowerName := strings.ToLower(fileName)
	
	if strings.HasSuffix(lowerName, ".ai") {
		return "AI"   // Adobe Illustrator
	} else if strings.HasSuffix(lowerName, ".psd") {
		return "PSD"  // Adobe Photoshop
	} else if strings.HasSuffix(lowerName, ".sketch") {
		return "SKETCH" // Sketch
	} else if strings.HasSuffix(lowerName, ".fig") {
		return "FIG"  // Figma
	} else if strings.HasSuffix(lowerName, ".xd") {
		return "XD"   // Adobe XD
	}
	return "FILE"  // Generic file
}