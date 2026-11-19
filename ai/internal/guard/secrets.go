package guard

import (
	"regexp"
	"strings"
)

const (
	RedactedText        = "[секрет приховано]"
	minimumSecretLength = 8
)

var secretPatterns = regexp.MustCompile(strings.Join([]string{
	`eyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`,
	`AIza[0-9A-Za-z_-]{35}`,
	`(?i)bearer\s+[A-Za-z0-9._~+/-]{16,}=*`,
}, "|"))

type SecretRedactor struct {
	secrets []string
}

func NewSecretRedactor(secrets ...string) *SecretRedactor {
	known := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		if len(secret) >= minimumSecretLength {
			known = append(known, secret)
		}
	}

	return &SecretRedactor{secrets: known}
}

func (redactor *SecretRedactor) Redact(text string) (string, bool) {
	redacted := text
	for _, secret := range redactor.secrets {
		redacted = strings.ReplaceAll(redacted, secret, RedactedText)
	}

	redacted = secretPatterns.ReplaceAllLiteralString(redacted, RedactedText)

	return redacted, redacted != text
}
