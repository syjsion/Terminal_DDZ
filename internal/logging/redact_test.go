package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestRedactingWriter(t *testing.T) {
	var output bytes.Buffer
	w := &RedactingWriter{Writer: &output, Secrets: []string{"sk-abcdefghijk"}}
	input := "provider failed with sk-abcdefghijk"
	if n, err := w.Write([]byte(input)); err != nil || n != len(input) {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if strings.Contains(output.String(), "sk-abcdefghijk") || !strings.Contains(output.String(), "sk-a...ijk") {
		t.Fatalf("secret not redacted: %q", output.String())
	}
}
