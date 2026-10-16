package fit

import (
	"bytes"
	"strings"
	"testing"
)

func TestStringValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in   string
		size int
		want string
	}{
		{"Finish", 16, "Finish"},
		{"Col de la Seigne", 16, "Col de la Seign"},
		// Multibyte characters aren't cut in half.
		{strings.Repeat("é", 10), 16, strings.Repeat("é", 7)},
	} {
		got := StringValue(tc.in, tc.size)
		if len(got) != tc.size || string(bytes.TrimRight(got, "\x00")) != tc.want {
			t.Errorf("StringValue(%q, %d) = %q, want %q", tc.in, tc.size, got, tc.want)
		}
	}
}
