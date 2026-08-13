package privacy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
)

const redactedPrivateKey = "[REDACTED:PRIVATE_KEY]"

// FilterText removes credential-shaped values before any event reaches storage.
// It intentionally preserves explanatory prose such as "store API keys securely".
func FilterText(input string) string {
	filtered := privateKeyPattern.ReplaceAllString(input, redactedPrivateKey)
	for _, rule := range boundedSecretPatterns {
		filtered = rule.pattern.ReplaceAllString(filtered, rule.replacement)
	}
	return highEntropyCandidatePattern.ReplaceAllStringFunc(filtered, func(candidate string) string {
		if looksHighEntropy(candidate) {
			return "[REDACTED:HIGH_ENTROPY]"
		}
		return candidate
	})
}

// FilterJSON applies the same pre-storage policy structurally so credential
// fields are removed even when their values do not resemble a known token.
func FilterJSON(input []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode event payload: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}
	return json.Marshal(filterJSONValue(value))
}

func filterJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		filtered := make(map[string]any, len(typed))
		for key, child := range typed {
			if sensitiveJSONKeyPattern.MatchString(key) {
				filtered[key] = "[REDACTED:SECRET]"
				continue
			}
			filtered[key] = filterJSONValue(child)
		}
		return filtered
	case []any:
		filtered := make([]any, len(typed))
		for index, child := range typed {
			filtered[index] = filterJSONValue(child)
		}
		return filtered
	case string:
		return FilterText(typed)
	default:
		return value
	}
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing event payload: %w", err)
	}
	return fmt.Errorf("event payload contains more than one JSON value")
}

func looksHighEntropy(value string) bool {
	if strings.HasPrefix(value, "[REDACTED:") || len(value) < 32 {
		return false
	}

	var lower, upper, digit, symbol bool
	counts := make(map[rune]int)
	for _, character := range value {
		counts[character]++
		switch {
		case character >= 'a' && character <= 'z':
			lower = true
		case character >= 'A' && character <= 'Z':
			upper = true
		case character >= '0' && character <= '9':
			digit = true
		default:
			symbol = true
		}
	}

	categories := 0
	for _, present := range []bool{lower, upper, digit, symbol} {
		if present {
			categories++
		}
	}
	if categories < 3 {
		return false
	}

	length := float64(len([]rune(value)))
	entropy := 0.0
	for _, count := range counts {
		probability := float64(count) / length
		entropy -= probability * math.Log2(probability)
	}
	return entropy >= 3.8
}
