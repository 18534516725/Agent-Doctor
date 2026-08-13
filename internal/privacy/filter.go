package privacy

import (
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
