package updates

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSelectRejectsUntrustedOrMismatchedArtifacts(t *testing.T) {
	valid := Artifact{Version: "1.1.0", OS: "darwin", Arch: "arm64", Filename: "agent-doctor_1.1.0_darwin_arm64.tar.gz", URL: "https://github.com/18534516725/Agent-Doctor/releases/download/v1.1.0/agent-doctor_1.1.0_darwin_arm64.tar.gz", Size: 12, SHA256: stringOf('a', 64)}
	for name, mutate := range map[string]func(*Artifact){
		"host":         func(item *Artifact) { item.URL = "https://attacker.example/file" },
		"filename":     func(item *Artifact) { item.Filename = "other.tar.gz" },
		"architecture": func(item *Artifact) { item.Arch = "amd64" },
		"version":      func(item *Artifact) { item.Version = "0.9.0" },
		"size":         func(item *Artifact) { item.Size = 0 },
		"sha":          func(item *Artifact) { item.SHA256 = "invalid" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if _, err := Select(Manifest{Version: candidate.Version, Artifacts: []Artifact{candidate}}, "1.0.0", "darwin", "arm64"); err == nil {
				t.Fatal("unsafe artifact was accepted")
			}
		})
	}
}

func TestVerifyArtifactChecksExactSizeAndDigest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact")
	raw := []byte("verified release")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	artifact := Artifact{Size: int64(len(raw)), SHA256: hex.EncodeToString(sum[:])}
	if err := VerifyArtifact(path, artifact); err != nil {
		t.Fatal(err)
	}
	artifact.Size++
	if err := VerifyArtifact(path, artifact); err == nil {
		t.Fatal("wrong size accepted")
	}
}

func TestInterruptedReplacementKeepsOldExecutableUsable(t *testing.T) {
	directory := t.TempDir()
	current := filepath.Join(directory, "agent-doctor")
	staged := filepath.Join(directory, "agent-doctor.new")
	if err := os.WriteFile(current, []byte("old executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staged, []byte("new executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("new executable"))
	originalRename := renameFile
	renameFile = func(old, next string) error {
		if old == staged {
			return errors.New("simulated interruption")
		}
		return os.Rename(old, next)
	}
	t.Cleanup(func() { renameFile = originalRename })
	if err := Apply(current, staged, Artifact{Size: int64(len("new executable")), SHA256: hex.EncodeToString(sum[:])}); err == nil {
		t.Fatal("interruption not reported")
	}
	stored, err := os.ReadFile(current)
	if err != nil || string(stored) != "old executable" {
		t.Fatalf("old executable lost: %q %v", stored, err)
	}
}

func stringOf(value byte, count int) string {
	result := make([]byte, count)
	for index := range result {
		result[index] = value
	}
	return string(result)
}
