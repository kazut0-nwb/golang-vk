package search

import (
	"strings"
	"unicode"
)

// Tokenize разбивает текст на уникальные слова.
// Словами считаются последовательности букв и цифр.
// Результат приводится к нижнему регистру.
func Tokenize(value string) []string {
	value = strings.ToLower(value)

	tokens := make([]string, 0)
	seen := make(map[string]struct{})

	var current strings.Builder

	addToken := func() {
		if current.Len() == 0 {
			return
		}

		token := current.String()
		current.Reset()

		if _, exists := seen[token]; exists {
			return
		}

		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}

	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			addToken()
		}
	}

	addToken()

	return tokens
}
