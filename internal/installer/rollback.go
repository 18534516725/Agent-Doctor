package installer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

func rollbackApplied(applied []appliedChange) error {
	var rollbackErrors []error
	for index := len(applied) - 1; index >= 0; index-- {
		item := applied[index]
		if item.existed {
			if err := atomicWrite(item.change.Path, item.change.Before, item.mode); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restore %s: %w", item.change.Path, err))
			}
		} else {
			if err := os.Remove(item.change.Path); err != nil && !os.IsNotExist(err) {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("remove %s: %w", item.change.Path, err))
			}
		}
	}
	return errors.Join(rollbackErrors...)
}

func withRollbackError(cause error, applied []appliedChange) error {
	if rollbackErr := rollbackApplied(applied); rollbackErr != nil {
		return errors.Join(cause, fmt.Errorf("transaction rollback incomplete: %w", rollbackErr))
	}
	return cause
}

func UninstallMarkedBlock(path, owner string) error {
	current, mode, exists, err := readCurrent(path)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	updated, found := removeMarkedBlock(current, owner)
	for found {
		var nextFound bool
		updated, nextFound = removeMarkedBlock(updated, owner)
		found = nextFound
	}
	if bytes.Equal(current, updated) {
		return nil
	}
	if err := atomicWrite(path, updated, mode); err != nil {
		return fmt.Errorf("remove Agent Doctor block: %w", err)
	}
	return nil
}

func removeMarkedBlock(contents []byte, owner string) ([]byte, bool) {
	begin := []byte(BeginMarker(owner))
	end := []byte(EndMarker(owner))
	start := bytes.Index(contents, begin)
	if start < 0 {
		return append([]byte(nil), contents...), false
	}
	finishRelative := bytes.Index(contents[start:], end)
	if finishRelative < 0 {
		return append([]byte(nil), contents...), false
	}
	finish := start + finishRelative + len(end)
	if finish < len(contents) && contents[finish] == '\r' {
		finish++
	}
	if finish < len(contents) && contents[finish] == '\n' {
		finish++
	}
	updated := append([]byte(nil), contents[:start]...)
	updated = append(updated, contents[finish:]...)
	return updated, true
}
