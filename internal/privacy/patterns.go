package privacy

import "regexp"

const sensitiveCredentialKey = `(?:password|passwd|pwd|api[_-]?key|access[_-]?token|refresh[_-]?token|session[_-]?token|client[_-]?secret|secret|token|authorization|cookie|set-cookie|aws[_-]?access[_-]?key[_-]?id|aws[_-]?secret[_-]?access[_-]?key|(?:[a-z0-9]+[_-])+(?:password|passwd|pwd|api[_-]?key|access[_-]?token|refresh[_-]?token|session[_-]?token|client[_-]?secret|secret|token))`

type replacementPattern struct {
	pattern     *regexp.Regexp
	replacement string
}

var boundedSecretPatterns = []replacementPattern{
	{
		pattern:     regexp.MustCompile(`(?im)\b(authorization\s*:\s*)(?:bearer|basic)\s+[^\r\n]+`),
		replacement: `${1}[REDACTED:AUTHORIZATION]`,
	},
	{
		pattern:     regexp.MustCompile(`(?im)\b(cookie|set-cookie)\s*:[^\r\n]*`),
		replacement: `${1}: [REDACTED:COOKIE]`,
	},
	{
		pattern: regexp.MustCompile(
			`(?i)\b(` + sensitiveCredentialKey + `)\s*([:=])\s*("[^"]*"|'[^']*'|[^\s,;&]+)`,
		),
		replacement: `${1}${2}[REDACTED:SECRET]`,
	},
	{
		pattern:     regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{8,}\b`),
		replacement: `[REDACTED:API_TOKEN]`,
	},
	{
		pattern:     regexp.MustCompile(`\b(?:gh[oprsu]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})\b`),
		replacement: `[REDACTED:API_TOKEN]`,
	},
}

var privateKeyPattern = regexp.MustCompile(`(?s)-----BEGIN(?: [A-Z0-9]+)? PRIVATE KEY-----.*?-----END(?: [A-Z0-9]+)? PRIVATE KEY-----`)

var highEntropyCandidatePattern = regexp.MustCompile(`\b[A-Za-z0-9_+/=-]{32,512}\b`)

var sensitiveJSONKeyPattern = regexp.MustCompile(`(?i)^` + sensitiveCredentialKey + `$`)
