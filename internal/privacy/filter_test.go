package privacy

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type corpusFixture struct {
	Blocked []string `json:"blocked"`
	Allowed []string `json:"allowed"`
}

func TestFilterCorpus(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/privacy/corpus.json")
	if err != nil {
		t.Fatal(err)
	}

	var corpus corpusFixture
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}

	for _, input := range corpus.Blocked {
		if got := FilterText(input); got == input || strings.Contains(got, secretValue(input)) {
			t.Fatalf("secret survived: %q => %q", input, got)
		}
	}
	for _, input := range corpus.Allowed {
		if got := FilterText(input); got != input {
			t.Fatalf("false positive: %q => %q", input, got)
		}
	}
}

func TestFilterTextRedactsMultilinePrivateKeys(t *testing.T) {
	input := "before\n-----BEGIN PRIVATE KEY-----\nsynthetic-private-material\n-----END PRIVATE KEY-----\nafter"
	got := FilterText(input)
	if strings.Contains(got, "synthetic-private-material") {
		t.Fatalf("private key survived: %q", got)
	}
}

func TestFilterTextRedactsHighEntropyAssignmentButNotExplanation(t *testing.T) {
	secret := "aB9vT2mK8zQ4xR7pL3sN6wY1cD5fG0hJ"
	got := FilterText("session_token=" + secret)
	if strings.Contains(got, secret) {
		t.Fatalf("high entropy assignment survived: %q", got)
	}

	explanation := "Session tokens should be stored securely."
	if got := FilterText(explanation); got != explanation {
		t.Fatalf("normal explanation changed: %q", got)
	}
}

func FuzzFilterNeverReturnsBearerValue(f *testing.F) {
	for _, value := range []string{"abc123", "synthetic-token-value", "aB9_7zY3-kLmN2"} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if value == "" || strings.ContainsAny(value, " \t\r\n") {
			return
		}
		got := FilterText("Authorization: Bearer " + value)
		if strings.Contains(got, "Bearer "+value) {
			t.Fatalf("bearer credential survived")
		}
	})
}

func secretValue(input string) string {
	for _, separator := range []string{"Bearer ", "password=", "Cookie: ", "api_key=", "token="} {
		if _, value, ok := strings.Cut(input, separator); ok {
			return strings.Trim(value, "\"'")
		}
	}
	return input
}
