package sheller

import (
	"strings"
)

// package sheller holds utilities that enable working with unix shells

// QuoteWrap will wrap the word in double-quotes and escape special characters
// as per:
// https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html#tag_18_02_03
func QuoteWrap(word string) string {
	strb := strings.Builder{}
	strb.WriteString(`"`)
	for _, r := range word {
		switch r {
		// Special characters inside double quotes:
		case '$', '`', '"', '\\', '\n':
			strb.WriteRune('\\')
		}
		strb.WriteRune(r)
	}
	strb.WriteString(`"`)
	return strb.String()
}
