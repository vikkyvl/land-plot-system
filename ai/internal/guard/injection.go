package guard

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const MaskedText = "[вилучено]"

var injectionPatterns = compilePatterns(
	`ignore\s+(all\s+|any\s+|the\s+)?(previous|prior|above|earlier)\s+(instructions|prompts?|rules|messages)`,
	`disregard\s+(all\s+|any\s+|the\s+)?(previous|prior|above|earlier)\s+(instructions|prompts?|rules)`,
	`forget\s+(all\s+|your\s+|the\s+)?(previous\s+)?(instructions|rules)`,
	`system\s+prompt`,
	`developer\s+(mode|message)`,
	`you\s+are\s+now`,
	`\breveal\b`,
	`api[\s_-]*key`,
	`access[\s_-]*token`,
	`ігноруй\s+(всі\s+|усі\s+)?(попередні|минулі|вищі)\s+(інструкції|вказівки|правила)`,
	`забудь\s+(всі\s+|усі\s+|свої\s+)?(попередні\s+)?(інструкції|вказівки|правила)`,
	`системн\S*\s+промпт`,
	`ти\s+тепер`,
	`покажи\s+(свій\s+|мені\s+)?(токен|ключ|пароль)`,
	`игнорируй\s+(все\s+)?(предыдущие|прошлые)\s+(инструкции|указания)`,
)

func compilePatterns(patterns ...string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		compiled = append(compiled, regexp.MustCompile(`(?i)`+pattern))
	}

	return compiled
}

func ContainsInjection(text string) bool {
	for _, pattern := range injectionPatterns {
		if pattern.MatchString(text) {
			return true
		}
	}

	return false
}

func MaskInjections(text string) (string, bool) {
	masked := text
	for _, pattern := range injectionPatterns {
		masked = pattern.ReplaceAllLiteralString(masked, MaskedText)
	}

	return masked, masked != text
}

func CheckUserInput(text string, maxChars int) error {
	text = strings.TrimSpace(text)

	switch {
	case text == "":
		return ErrEmptyInput
	case utf8.RuneCountInString(text) > maxChars:
		return fmt.Errorf(formatInputTooLong, ErrInputTooLong, maxChars)
	case ContainsInjection(text):
		return ErrSuspiciousInput
	}

	return nil
}
