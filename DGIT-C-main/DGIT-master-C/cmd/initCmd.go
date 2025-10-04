package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	
	initializer "dgit/internal/init"
	"github.com/spf13/cobra"
)

// InitCmd represents the init command for initializing a new DGit repository
// Similar to 'git init' but creates DGit-specific directory structure
var InitCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new DGit repository",
	Long: `Initialize a new DGit repository in the specified directory.
If no directory is specified, initializes in the current directory.

This creates a .dgit folder with the necessary repository structure.`,
	Args: cobra.MaximumNArgs(1),  // Optional directory argument
	Run:  runInit,
}

// runInit executes the init command functionality
// Creates the .dgit directory structure and necessary files for a new repository
func runInit(cmd *cobra.Command, args []string) {
	// Determine target directory for initialization
	var targetDir string
	if len(args) == 0 {
		// No directory specified, use current directory
		targetDir = "."
	} else {
		// Use specified directory
		targetDir = args[0]
	}

	// Initialize the repository using the internal initializer
	initMgr := initializer.NewRepositoryInitializer()
	if err := initMgr.InitializeRepository(targetDir); err != nil {
		printError(fmt.Sprintf("%v", err))
		os.Exit(1)
	}

	// Display success message with absolute path
	absPath, _ := filepath.Abs(targetDir)
	printSuccess(fmt.Sprintf("Initialized DGit repository in %s", absPath))
}