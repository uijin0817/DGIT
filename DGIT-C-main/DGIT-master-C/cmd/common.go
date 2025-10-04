package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/fatih/color"
)

// Color functions using fatih/color library for better compatibility
var (
	// Define color functions
	red     = color.New(color.FgRed).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	bold    = color.New(color.Bold).SprintFunc()
	
	// Print functions
	printRed    = color.New(color.FgRed).PrintlnFunc()
	printGreen  = color.New(color.FgGreen).PrintlnFunc()
	printYellow = color.New(color.FgYellow).PrintlnFunc()
	printBlue   = color.New(color.FgBlue).PrintlnFunc()
	printCyan   = color.New(color.FgCyan).PrintlnFunc()
	printBold   = color.New(color.Bold).PrintlnFunc()
)

// isInDgitRepository checks if the current directory is within a DGit repository
// Returns true if .dgit directory is found in current or parent directories
func isInDgitRepository() bool {
	return findDgitDirectory() != ""
}

// findDgitDirectory finds the .dgit directory by traversing up the directory tree
// Similar to how Git finds .git directory - searches from current dir up to root
func findDgitDirectory() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Traverse up the directory tree looking for .dgit folder
	for {
		dgitPath := filepath.Join(currentDir, ".dgit")
		if info, err := os.Stat(dgitPath); err == nil && info.IsDir() {
			return dgitPath
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached root directory
		}
		currentDir = parent
	}

	return ""
}

// checkDgitRepository checks if we're in a DGit repository and exits with error message if not
// Convenience function that combines check and error handling
func checkDgitRepository() string {
	if !isInDgitRepository() {
		exitWithError("not a dgit repository (or any of the parent directories)", "Run 'dgit init' to initialize a repository")
	}
	return findDgitDirectory()
}

// exitWithError prints error messages and exits with status code 1
// Provides consistent error handling across all commands
func exitWithError(message string, suggestion string) {
	if message != "" {
		if suggestion != "" {
			// Error with helpful suggestion
			printError(message)
			printSuggestion(suggestion)
		} else {
			// Simple error message
			printError(message)
		}
	}
	os.Exit(1)
}

// printError prints an error message with red color formatting
func printError(message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", red("Error"), message)
}

// printSuggestion prints a suggestion message with yellow color formatting
func printSuggestion(message string) {
	fmt.Fprintf(os.Stderr, "%s\n", yellow(message))
}

// printSuccess prints a success message with green color formatting
func printSuccess(message string) {
	fmt.Printf("%s %s\n", green("✓"), message)
}

// printWarning prints a warning message with yellow color formatting
func printWarning(message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", yellow("Warning"), message)
}

// printInfo prints an informational message with default color
func printInfo(message string) {
	fmt.Println(message)
}

// Helper functions for colored output using fatih/color library
// These functions provide convenient access to colored printing