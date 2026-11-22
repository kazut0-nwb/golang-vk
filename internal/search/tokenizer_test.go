package search

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "basic tokenization",
			input: "Solaris ANDREI Tarkovsky",
			want:  []string{"solaris", "andrei", "tarkovsky"},
		},
		{
			name:  "uppercase and punctuation",
			input: "SOLARIS, Solaris! TARKOVSKY.",
			want:  []string{"solaris", "tarkovsky"},
		},
		{
			name:  "digits",
			input: "Movie 1972 Part 2",
			want:  []string{"movie", "1972", "part", "2"},
		},
		{
			name:  "unicode",
			input: "КИНО-2026 café",
			want:  []string{"кино", "2026", "café"},
		},
		{
			name:  "empty input",
			input: "   !!! ",
			want:  []string{},
		},
		{
			name:  "duplicate words",
			input: "Solaris solaris SOLARIS",
			want:  []string{"solaris"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tokenize(tt.input)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"Tokenize(%q) = %v, want %v",
					tt.input,
					got,
					tt.want,
				)
			}
		})
	}
}
