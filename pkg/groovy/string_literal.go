package groovy

import (
	"fmt"
	"strings"
)

// StringLiteral quotes a value as a non-interpolating Groovy string.
func StringLiteral(value string) string {
	var literal strings.Builder
	literal.WriteByte('\'')
	for _, r := range value {
		switch r {
		case '\'', '\\':
			literal.WriteByte('\\')
			literal.WriteRune(r)
		case '\b':
			literal.WriteString(`\b`)
		case '\f':
			literal.WriteString(`\f`)
		case '\n':
			literal.WriteString(`\n`)
		case '\r':
			literal.WriteString(`\r`)
		case '\t':
			literal.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f || r == 0x85 || r == 0x2028 || r == 0x2029 {
				fmt.Fprintf(&literal, `\u%04x`, r)
			} else {
				literal.WriteRune(r)
			}
		}
	}
	literal.WriteByte('\'')
	return literal.String()
}
