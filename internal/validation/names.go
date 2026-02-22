package validation

import (
	"regexp"
	"strings"
)

const (
	minCharacterNameLength = 2
	maxCharacterNameLength = 29
	minGuildNameLength     = 3
	maxGuildNameLength     = 29
)

var validNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z' -]*[A-Za-z]$`)

func IsCharacterNameValid(input string) (string, error) {
	return validateName(
		input,
		minCharacterNameLength,
		maxCharacterNameLength,
		ErrorCharacterNameEmpty,
		ErrorCharacterNameTooShort,
		ErrorCharacterNameTooLong,
		ErrorCharacterNameRepeatedSpaces,
		ErrorCharacterNameInvalidBoundary,
		ErrorCharacterNameInvalidSymbols,
		ErrorCharacterNameInvalidFormat,
	)
}

func IsGuildNameValid(input string) (string, error) {
	return validateName(
		input,
		minGuildNameLength,
		maxGuildNameLength,
		ErrorGuildNameEmpty,
		ErrorGuildNameTooShort,
		ErrorGuildNameTooLong,
		ErrorGuildNameRepeatedSpaces,
		ErrorGuildNameInvalidBoundary,
		ErrorGuildNameInvalidSymbols,
		ErrorGuildNameInvalidFormat,
	)
}

func normalizeNameValue(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}

func normalizeLookupValue(input string) string {
	normalized := strings.ToLower(normalizeNameValue(strings.ReplaceAll(input, "+", " ")))
	normalized = strings.ReplaceAll(normalized, "'", "")
	return normalized
}

func validateName(
	input string,
	minLen int,
	maxLen int,
	errEmpty int,
	errShort int,
	errLong int,
	errRepeatedSpaces int,
	errInvalidBoundary int,
	errInvalidSymbols int,
	errInvalidFormat int,
) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", NewError(errEmpty, "name cannot be empty", nil)
	}

	normalized := normalizeNameValue(trimmed)
	if len(normalized) < minLen {
		return "", NewError(errShort, "name is too short", nil)
	}
	if len(normalized) > maxLen {
		return "", NewError(errLong, "name is too long", nil)
	}

	first := normalized[0]
	last := normalized[len(normalized)-1]
	if !isLetter(first) || !isLetter(last) {
		return "", NewError(errInvalidBoundary, "name must start and end with a letter", nil)
	}

	if !validNamePattern.MatchString(normalized) {
		for _, char := range normalized {
			if isAllowedNameCharacter(char) {
				continue
			}
			return "", NewError(errInvalidSymbols, "name has unsupported characters", nil)
		}
		return "", NewError(errInvalidFormat, "name format is invalid", nil)
	}

	return normalized, nil
}

func isLetter(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

func isAllowedNameCharacter(char rune) bool {
	if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
		return true
	}
	return char == '\'' || char == '-' || char == ' '
}
