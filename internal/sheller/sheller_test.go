package sheller

import (
	"testing"
)

func TestQuoteWrap(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		// regular string
		{
			input:    "Hello",
			expected: `"Hello"`,
		},
		// embedded double-quotes
		{
			input:    `Hello "World"`,
			expected: `"Hello \"World\""`,
		},
		// embedded backtick-quotes
		{
			input:    "Hello `World`",
			expected: "\"Hello \\`World\\`\"",
		},
		// embedded single-quotes,
		{
			input:    "Hello 'World'",
			expected: "\"Hello 'World'\"",
		},
		// escaping $, backslash (\), and \n
		{
			input:    `A $pecial charac\ter\n`,
			expected: `"A \$pecial charac\\ter\\n"`,
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.input, func(t *testing.T) {
				result := QuoteWrap(tc.input)
				if result != tc.expected {
					t.Errorf("Expected %q, but got %q", tc.expected, result)
				}
			},
		)
	}
}
