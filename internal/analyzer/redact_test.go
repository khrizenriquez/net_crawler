package analyzer

import (
	"strings"
	"testing"
)

func TestRedactNeverKeepsSecretsOrBase64Payload(t *testing.T) {
	input := "user=hello@example.test&password=hunter2 data:image/png;base64,QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo="
	got := Redact(input)
	for _, forbidden := range []string{"hunter2", "hello@example.test", "QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo="} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("redacted output kept %q: %s", forbidden, got)
		}
	}
}

func TestRedactCoversCommonSecretNames(t *testing.T) {
	got := Redact("passwd=a pwd=b pass=c token=d secret=e")
	for _, forbidden := range []string{"passwd=a", "pwd=b", "pass=c", "token=d", "secret=e"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("redacted output kept %q: %s", forbidden, got)
		}
	}
}

func TestLooksSensitiveAndHashContent(t *testing.T) {
	for _, sample := range []string{"Password=x", "Authorization: Basic abc", "TOKEN=x", "user=alice"} {
		if !LooksSensitive(sample) {
			t.Fatalf("expected %q to look sensitive", sample)
		}
	}
	if LooksSensitive("ordinary plaintext") {
		t.Fatal("ordinary plaintext should not look sensitive")
	}
	if got := HashContent("same"); got == "" || got != HashContent("same") || got == HashContent("different") {
		t.Fatalf("unexpected hashes: %q %q", got, HashContent("different"))
	}
}
