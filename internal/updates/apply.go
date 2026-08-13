package updates

import (
	"fmt"
	"os"
)

var renameFile = os.Rename

// Apply transactionally replaces only the Agent Doctor executable. It never
// searches for, stops, restarts or signals an AI client process.
func Apply(currentPath, stagedPath string, artifact Artifact) error {
	if err := VerifyArtifact(stagedPath, artifact); err != nil {
		return err
	}
	backupPath := currentPath + ".previous-update"
	if _, err := os.Stat(backupPath); err == nil {
		return fmt.Errorf("previous update backup already exists")
	}
	if err := renameFile(currentPath, backupPath); err != nil {
		return fmt.Errorf("stage current executable: %w", err)
	}
	if err := renameFile(stagedPath, currentPath); err != nil {
		if rollbackErr := renameFile(backupPath, currentPath); rollbackErr != nil {
			return fmt.Errorf("install update: %w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("install update: %w", err)
	}
	if err := os.Chmod(currentPath, 0o755); err != nil {
		return fmt.Errorf("set updated executable permissions: %w", err)
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("remove update backup: %w", err)
	}
	return nil
}
