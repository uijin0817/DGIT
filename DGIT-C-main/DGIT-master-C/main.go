package main

import (
	"fmt"
	"os"

	"dgit/cmd"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dgit",
	Short: "DGit - Version Control for Design Files",
	Long: `DGit is a specialized version control system for design files.
It provides intelligent tracking for Adobe Illustrator, Photoshop, Sketch, 
Figma, and other design file formats with metadata-aware versioning.

🎨 Key Features:
- Metadata-aware version tracking for design files
- Design file format support (AI, PSD, Sketch, Figma, XD)
- Visual diff for design changes with layer/artboard tracking
- Team collaboration optimized for creative workflows
- Git-like interface with design-specific enhancements`,
}

func init() {
	rootCmd.AddCommand(cmd.InitCmd)
	rootCmd.AddCommand(cmd.AddCmd)
	rootCmd.AddCommand(cmd.CommitCmd)
	rootCmd.AddCommand(cmd.StatusCmd)
	rootCmd.AddCommand(cmd.LogCmd)
	rootCmd.AddCommand(cmd.RestoreCmd)
	rootCmd.AddCommand(cmd.ScanCmd)
	rootCmd.AddCommand(cmd.ShowCmd) // 새로 추가
}
func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
}
