package installer

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ApplyResult struct {
	Applied         int    `json:"applied"`
	Skipped         int    `json:"skipped"`
	BackupDirectory string `json:"backupDirectory,omitempty"`
}

type appliedChange struct {
	change  Change
	existed bool
	mode    os.FileMode
}

type backupManifestEntry struct {
	Path    string      `json:"path"`
	Existed bool        `json:"existed"`
	Mode    os.FileMode `json:"mode"`
	Backup  string      `json:"backup,omitempty"`
}

func Apply(plan Plan) (ApplyResult, error) {
	result := ApplyResult{}
	applied := make([]appliedChange, 0, len(plan.Changes))
	backupDirectory := ""
	manifest := make([]backupManifestEntry, 0, len(plan.Changes))

	for _, change := range plan.Changes {
		current, mode, existed, err := readCurrent(change.Path)
		if err != nil {
			return ApplyResult{}, withRollbackError(fmt.Errorf("read %s before apply: %w", change.Path, err), applied)
		}
		if bytes.Equal(current, change.After) {
			result.Skipped++
			continue
		}
		if !bytes.Equal(current, change.Before) {
			return ApplyResult{}, withRollbackError(fmt.Errorf("configuration changed after planning: %s", change.Path), applied)
		}
		if backupDirectory == "" {
			backupDirectory, err = newBackupDirectory(plan.Home)
			if err != nil {
				return ApplyResult{}, withRollbackError(err, applied)
			}
			result.BackupDirectory = backupDirectory
		}
		entry := backupManifestEntry{Path: change.Path, Existed: existed, Mode: mode}
		if existed {
			entry.Backup = filepath.Join(backupDirectory, fmt.Sprintf("%03d.before", len(manifest)+1))
			if err := os.WriteFile(entry.Backup, current, 0o600); err != nil {
				return ApplyResult{}, withRollbackError(fmt.Errorf("write installer backup: %w", err), applied)
			}
		}
		manifest = append(manifest, entry)
		if err := atomicWrite(change.Path, change.After, change.Mode); err != nil {
			return ApplyResult{}, withRollbackError(fmt.Errorf("apply %s: %w", change.Path, err), applied)
		}
		applied = append(applied, appliedChange{change: change, existed: existed, mode: mode})
		result.Applied++
	}
	if backupDirectory != "" {
		if err := writeBackupManifest(backupDirectory, manifest); err != nil {
			return ApplyResult{}, withRollbackError(err, applied)
		}
	}
	return result, nil
}

func newBackupDirectory(home string) (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate backup id: %w", err)
	}
	directory := filepath.Join(home, ".agent-doctor", "backups", time.Now().UTC().Format("20060102T150405.000000000Z")+"-"+hex.EncodeToString(random))
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create installer backup directory: %w", err)
	}
	return directory, nil
}

func writeBackupManifest(directory string, entries []backupManifestEntry) error {
	raw, err := json.MarshalIndent(map[string]any{"schemaVersion": 1, "changes": entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode installer backup manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), raw, 0o600); err != nil {
		return fmt.Errorf("write installer backup manifest: %w", err)
	}
	return nil
}

func readCurrent(path string) ([]byte, os.FileMode, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, 0o600, false, nil
	}
	if err != nil {
		return nil, 0, false, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, false, err
	}
	return raw, info.Mode().Perm(), true, nil
}

func atomicWrite(path string, contents []byte, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o600
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".agent-doctor-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode.Perm()); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err == nil {
		return nil
	}
	return replacePortable(temporaryPath, path)
}

func replacePortable(temporaryPath, targetPath string) error {
	oldFile, err := os.CreateTemp(filepath.Dir(targetPath), ".agent-doctor-old-*")
	if err != nil {
		return err
	}
	oldPath := oldFile.Name()
	if err := oldFile.Close(); err != nil {
		return err
	}
	if err := os.Remove(oldPath); err != nil {
		return err
	}
	defer os.Remove(oldPath)
	targetExisted := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExisted = true
		if err := os.Rename(targetPath, oldPath); err != nil {
			return err
		}
	}
	if err := os.Rename(temporaryPath, targetPath); err != nil {
		if targetExisted {
			_ = os.Rename(oldPath, targetPath)
		}
		return err
	}
	if targetExisted {
		_ = os.Remove(oldPath)
	}
	return nil
}
