package lexer

import "testing"

func TestIsKeyword(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"class", true},
		{"Class", true}, // case-insensitive
		{"CLASS", true},
		{"function", true},
		{"UserController", false},
		{"foo", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := IsKeyword(tc.in); got != tc.want {
				t.Errorf("IsKeyword(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
