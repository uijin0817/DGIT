package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"dgit/internal/log"
	"dgit/internal/restore"

	"github.com/spf13/cobra"
)

// RestoreCmd restores files from a specific commit
var RestoreCmd = &cobra.Command{
	Use:   "restore <version_or_hash> [file...]",
	Short: "Restore files from a specific commit",
	Long: `Restore files from a specific commit version or hash to the working directory.
If no files are specified, all files from that commit will be restored.

Examples:
  dgit restore 1                  # Restore all files from version 1
  dgit restore c3a5f7b8           # Restore all files from commit hash
  dgit restore 2 my_design.psd    # Restore specific file from version 2
  dgit restore 2 designs/         # Restore directory from version 2

File matching supports:
- Exact path matching
- Filename-only matching  
- Directory matching
- Partial path matching`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires at least one argument: <version_or_hash>")
		}
		return nil
	},
	Run: runRestore,
}

// runRestore restores files from a specific commit to the working directory
func runRestore(cmd *cobra.Command, args []string) {
	dgitDir := checkDgitRepository()

	restoreManager := restore.NewRestoreManager(dgitDir)
	logManager := log.NewLogManager(dgitDir)

	commitRef := args[0]
	filesToRestore := []string{}

	if len(args) > 1 {
		filesToRestore = args[1:]
	}

	targetCommit, err := findTargetCommit(logManager, commitRef)
	if err != nil {
		printError(fmt.Sprintf("Failed to find commit: %v", err))
		os.Exit(1)
	}

	if len(filesToRestore) == 0 {
		fmt.Printf("Restoring all files from commit %s (v%d)\n", targetCommit.Hash[:8], targetCommit.Version)
		fmt.Printf("\"%s\"\n", targetCommit.Message)
		fmt.Printf("Files: %d\n\n", targetCommit.FilesCount)
	} else {
		fmt.Printf("Restoring %d specific files from commit %s (v%d)\n", len(filesToRestore), targetCommit.Hash[:8], targetCommit.Version)
		fmt.Printf("\"%s\"\n", targetCommit.Message)
		fmt.Printf("Target files: %v\n\n", filesToRestore)
	}

	err = performRestore(restoreManager, targetCommit, filesToRestore)
	if err != nil {
		printError(fmt.Sprintf("Restore failed: %v", err))
		os.Exit(1)
	}
}

// findTargetCommit finds a commit by hash or version number
func findTargetCommit(logManager *log.LogManager, commitRef string) (*log.Commit, error) {
	var targetCommit *log.Commit
	var err error

	isHashCandidate := false
	if len(commitRef) >= 4 && len(commitRef) <= 64 {
		isHashCandidate = true
		for _, r := range commitRef {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				isHashCandidate = false
				break
			}
		}
	}

	if isHashCandidate {
		targetCommit, err = logManager.GetCommitByHash(commitRef)
		if err == nil && targetCommit != nil {
			return targetCommit, nil
		}
	}

	strippedCommitRef := strings.TrimPrefix(commitRef, "v")
	version, err := strconv.Atoi(strippedCommitRef)
	if err == nil {
		targetCommit, err = logManager.GetCommit(version)
		if err == nil && targetCommit != nil {
			return targetCommit, nil
		}
	}

	return nil, fmt.Errorf("commit '%s' not found", commitRef)
}

// performRestore performs the actual file restoration
func performRestore(restoreManager *restore.RestoreManager, targetCommit *log.Commit, filesToRestore []string) error {
	commitRef := fmt.Sprintf("v%d", targetCommit.Version)

	err := restoreManager.RestoreFilesFromCommit(commitRef, filesToRestore, targetCommit)

	return err
}
