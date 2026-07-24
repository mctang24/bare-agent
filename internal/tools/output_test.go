package tools

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestLimitedOutputPreservesUTF8WhenTruncated(t *testing.T) {
	markerPattern := regexp.MustCompile(`\[\.\.\. truncated ([0-9]+) bytes \.\.\.\]`)

	for prefixLength := range 3 {
		t.Run(strconv.Itoa(prefixLength), func(t *testing.T) {
			input := strings.Repeat("a", prefixLength) + strings.Repeat("中", toolOutputLimit)
			var output limitedOutput
			if _, err := output.Write([]byte(input)); err != nil {
				t.Fatal(err)
			}

			result := output.String()
			if !utf8.ValidString(result) {
				t.Fatal("output is not valid UTF-8")
			}
			match := markerPattern.FindStringSubmatch(result)
			if match == nil {
				t.Fatal("output does not contain the truncation marker")
			}
			omitted, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatal(err)
			}
			markerLength := len(match[0]) + 4
			if kept := len(result) - markerLength; kept > toolOutputLimit {
				t.Fatalf("kept output length = %d, want at most %d", kept, toolOutputLimit)
			}
			if want := len(input) - (len(result) - markerLength); omitted != want {
				t.Fatalf("omitted bytes = %d, want %d", omitted, want)
			}
		})
	}
}
