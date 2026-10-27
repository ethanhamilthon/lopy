package limitbuf

import (
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAllWriterPathsAreBoundedAndKeepHeadAndTail(t *testing.T) {
	payload := "HEAD-MARKER\n" + strings.Repeat("x", 2*Limit) + "\nTAIL-MARKER"
	for _, write := range []func(*Buffer){
		func(b *Buffer) { _, _ = b.Write([]byte(payload)) },
		func(b *Buffer) { _, _ = io.WriteString(b, payload) },
		func(b *Buffer) { _, _ = io.Copy(b, strings.NewReader(payload)) },
	} {
		b := &Buffer{}
		write(b)
		got := b.Text()
		if len(got) > Limit+100 || !strings.Contains(got, "[output truncated at 16 KiB") {
			t.Fatalf("cap bypassed: len=%d", len(got))
		}
		if !strings.HasPrefix(got, "HEAD-MARKER\n") || !strings.HasSuffix(got, "\nTAIL-MARKER") {
			t.Fatalf("head or tail lost: head=%q tail=%q", got[:20], got[len(got)-20:])
		}
		b.Reset()
		_, _ = b.Write([]byte("clean"))
		if b.Text() != "clean" {
			t.Fatal("reset retained truncation state")
		}
	}
	b := &Buffer{}
	_, _ = io.WriteString(b, strings.Repeat("界", Limit))
	if !utf8.ValidString(b.Text()) {
		t.Fatal("invalid UTF-8 after truncation")
	}
}
