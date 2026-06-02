package analyzer

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	passwordRE = regexp.MustCompile(`(?i)(password|passwd|pwd|pass|token|secret)\s*([=:]\s*)([^&\s;]+)`)
	emailRE    = regexp.MustCompile(`([A-Za-z0-9._%+\-])[^@\s]*(@[A-Za-z0-9.\-]+\.[A-Za-z]{2,})`)
	base64RE   = regexp.MustCompile(`(?i)(data:[a-z0-9.+\-]+/[a-z0-9.+\-]+;base64,)[A-Za-z0-9+/=]{24,}`)
)

func Redact(sample string) string {
	out := passwordRE.ReplaceAllString(sample, `$1$2[REDACTED]`)
	out = emailRE.ReplaceAllString(out, `$1***$2`)
	out = base64RE.ReplaceAllString(out, `$1[PAYLOAD OMITTED]`)
	return out
}

func HashContent(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func LooksSensitive(sample string) bool {
	lower := strings.ToLower(sample)
	for _, marker := range []string{"password=", "passwd=", "pwd=", "authorization:", "token=", "secret=", "user="} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
