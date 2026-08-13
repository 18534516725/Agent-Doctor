package updates

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func Select(manifest Manifest, currentVersion, targetOS, targetArch string) (Artifact, error) {
	if compareVersions(manifest.Version, currentVersion) <= 0 {
		return Artifact{}, fmt.Errorf("release version must be newer than the running version")
	}
	for _, artifact := range manifest.Artifacts {
		if artifact.OS != targetOS || artifact.Arch != targetArch {
			continue
		}
		if artifact.Version != manifest.Version || artifact.Size <= 0 || !validDigest(artifact.SHA256) {
			return Artifact{}, fmt.Errorf("artifact metadata is invalid")
		}
		extension := ".tar.gz"
		if targetOS == "windows" {
			extension = ".zip"
		}
		expectedFilename := fmt.Sprintf("agent-doctor_%s_%s_%s%s", artifact.Version, targetOS, targetArch, extension)
		if artifact.Filename != expectedFilename {
			return Artifact{}, fmt.Errorf("artifact filename does not match the target")
		}
		parsed, err := url.Parse(artifact.URL)
		expectedPrefix := "/18534516725/Agent-Doctor/releases/download/v" + artifact.Version + "/"
		if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "github.com") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != expectedPrefix+artifact.Filename {
			return Artifact{}, fmt.Errorf("artifact URL is not an approved GitHub release URL")
		}
		return artifact, nil
	}
	return Artifact{}, fmt.Errorf("no release artifact matches %s/%s", targetOS, targetArch)
}

func VerifyArtifact(path string, artifact Artifact) error {
	if artifact.Size <= 0 || !validDigest(artifact.SHA256) {
		return fmt.Errorf("artifact metadata is invalid")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open staged artifact: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect staged artifact: %w", err)
	}
	if info.Size() != artifact.Size {
		return fmt.Errorf("artifact size mismatch")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("hash staged artifact: %w", err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(actual), []byte(strings.ToLower(artifact.SHA256))) != 1 {
		return fmt.Errorf("artifact SHA-256 mismatch")
	}
	return nil
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func compareVersions(left, right string) int {
	leftParts, leftOK := parseVersion(left)
	rightParts, rightOK := parseVersion(right)
	if !leftOK || !rightOK {
		return -1
	}
	for index := range leftParts {
		if leftParts[index] > rightParts[index] {
			return 1
		}
		if leftParts[index] < rightParts[index] {
			return -1
		}
	}
	return 0
}

func parseVersion(value string) ([3]int, bool) {
	match := versionPattern.FindStringSubmatch(value)
	if match == nil {
		return [3]int{}, false
	}
	var result [3]int
	for index := range result {
		parsed, err := strconv.Atoi(match[index+1])
		if err != nil {
			return [3]int{}, false
		}
		result[index] = parsed
	}
	return result, true
}
