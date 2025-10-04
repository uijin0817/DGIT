package cmd

import (
	"fmt"
	"os"

	"dgit/internal/log"

	"github.com/spf13/cobra"
)

// LogCmd shows commit history with design-specific metadata
var LogCmd = &cobra.Command{
	Use:   "log",
	Short: "Show commit history",
	Long: `Display the commit history showing:
- Commit hashes and messages
- Author and timestamp information
- File counts and metadata summaries

Examples:
  dgit log                    # Show all commits
  dgit log --oneline          # Show compact format
  dgit log -n 5               # Show last 5 commits`,
	Run: runLog,
}

func init() {
	LogCmd.Flags().BoolP("oneline", "o", false, "Show commits in compact one-line format")
	LogCmd.Flags().IntP("number", "n", 0, "Limit the number of commits to show")
}

// runLog displays commit history with design-specific information
func runLog(cmd *cobra.Command, _ []string) {
	dgitDir := checkDgitRepository()
	logManager := log.NewLogManager(dgitDir)

	commits, err := logManager.GetCommitHistory()
	if err != nil {
		printError(fmt.Sprintf("loading commit history: %v", err))
		os.Exit(1)
	}

	if len(commits) == 0 {
		fmt.Println("No commits yet.")
		printInfo("Use 'dgit add' and 'dgit commit' to create your first commit.")
		return
	}

	oneline, _ := cmd.Flags().GetBool("oneline")
	number, _ := cmd.Flags().GetInt("number")

	if number > 0 && number < len(commits) {
		commits = commits[:number]
	}

	fmt.Printf("Commit History (%d commits)\n\n", len(commits))

	for i, c := range commits {
		if oneline {
			fmt.Printf("%s (v%d) %s\n", c.Hash[:8], c.Version, c.Message)
		} else {
			fmt.Printf("commit %s (v%d)\n", c.Hash[:12], c.Version)
			fmt.Printf("Author: %s\n", c.Author)
			fmt.Printf("Date: %s\n", c.Timestamp.Format("Mon Jan 2 15:04:05 2006"))
			fmt.Printf("\n    %s\n", c.Message)

			if c.FilesCount > 0 {
				fmt.Printf("    Files: %d", c.FilesCount)
				if c.SnapshotZip != "" {
					fmt.Printf(" (snapshot: %s)", c.SnapshotZip)
				}
				fmt.Println()

				summary := logManager.GenerateCommitSummary(c)
				if summary != fmt.Sprintf("[v%d] %s (%d files)", c.Version, c.Message, c.FilesCount) {
					fmt.Printf("    %s\n", summary)
				}
			}

			if i < len(commits)-1 {
				fmt.Println()
			}
		}
	}

	fmt.Printf("\nTotal: %d commits in history\n", len(commits))
}
