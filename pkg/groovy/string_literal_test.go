package groovy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringLiteral(t *testing.T) {
	for _, tt := range []struct{ name, value, want string }{
		{"empty", "", `''`},
		{"plain", "jobs/*.groovy", `'jobs/*.groovy'`},
		{"quotes", `it's "quoted"`, `'it\'s "quoted"'`},
		{"backslashes", `a\\b\`, `'a\\\\b\\'`},
		{"line endings", "a\r\nb\tc", `'a\r\nb\tc'`},
		{"controls", "\x00\b\f\x1b\x1f\x7f", `'\u0000\b\f\u001b\u001f\u007f'`},
		{"unicode", "caffè", `'caffè'`},
		{"unicode line separators", "\u0085\u2028\u2029", `'\u0085\u2028\u2029'`},
		{"literal unicode escapes", `\u0027\u000a\u005c`, `'\\u0027\\u000a\\u005c'`},
		{"dollar variables", "$HOME ${name}", `'$HOME ${name}'`},
		{"interpolation", `${1 + 1}`, `'${1 + 1}'`},
		{"breakout", `'); println('hello'); //`, `'\'); println(\'hello\'); //'`},
	} {
		t.Run(tt.name, func(t *testing.T) { require.Equal(t, tt.want, StringLiteral(tt.value)) })
	}
}
