package cmd

import (
	"fmt"
	"os"
	"strings"

	"dgit/internal/scanner"
	"github.com/spf13/cobra"
)

// ScanCmd scans directories for design files
var ScanCmd = &cobra.Command{
	Use:   "scan [folder]",
	Short: "Scan for design files",
	Long: `Discover design files in the specified folder.
Shows file count, types, and total size.

Examples:
  dgit scan           # Scan current directory
  dgit scan designs/  # Scan specific folder`,
	Args: cobra.MaximumNArgs(1),
	Run:  runScan,
}

// runScan performs scanning of design files
func runScan(cmd *cobra.Command, args []string) {
	var targetDir string
	if len(args) == 0 {
		targetDir = "."
	} else {
		targetDir = args[0]
	}

	if !isInDgitRepository() {
		printError("not a dgit repository (or any of the parent directories)")
		printSuggestion("Run 'dgit init' to initialize a repository")
		os.Exit(1)
	}

	fmt.Printf("Scanning design files in: %s\n", targetDir)

	quickScanner := scanner.NewQuickScanner()
	result, err := quickScanner.Scan(targetDir)
	if err != nil {
		printError(fmt.Sprintf("%v", err))
		os.Exit(1)
	}

	printScanResults(result)
}

// printScanResults displays scan results
func printScanResults(result *scanner.QuickScanResult) {
	if result.TotalFiles == 0 {
		fmt.Println("No design files found.")
		fmt.Println("   Supported formats: .ai, .psd, .sketch, .fig, .xd, .afdesign, .afphoto")
		return
	}

	fmt.Printf("✓ Found %d design files (%.1f MB)\n",
		result.TotalFiles, float64(result.TotalSize)/(1024*1024))

	if len(result.TypeCounts) > 0 {
		for fileType, count := range result.TypeCounts {
			fmt.Printf("   • %d %s files\n", count, strings.ToUpper(fileType))
		}
	}

	fmt.Printf("\nScan completed in %v\n", result.ScanTime)
	fmt.Println("Use 'dgit show <filename>' for detailed file analysis")
}
