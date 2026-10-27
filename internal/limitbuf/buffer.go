// Package limitbuf captures bounded UTF-8 tool output, keeping head and tail when truncated.
package limitbuf

import "strings"

const (
	Limit = 16 * 1024
	half  = Limit / 2
)

type Buffer struct {
	head, tail []byte
	size       int
}

func (b *Buffer) Write(p []byte) (int, error) {
	total := len(p)
	b.size += total
	if len(b.head) < half {
		n := min(len(p), half-len(b.head))
		b.head = append(b.head, p[:n]...)
		p = p[n:]
	}
	if len(p) >= half {
		b.tail = append(b.tail[:0], p[len(p)-half:]...)
	} else if len(p) > 0 {
		b.tail = append(b.tail, p...)
		if len(b.tail) > half {
			b.tail = append(b.tail[:0], b.tail[len(b.tail)-half:]...)
		}
	}
	return total, nil
}
func (b *Buffer) Reset() { b.head, b.tail, b.size = b.head[:0], b.tail[:0], 0 }
func (b *Buffer) Text() string {
	if b.size <= Limit {
		return strings.ToValidUTF8(string(b.head)+string(b.tail), "")
	}
	head, tail := strings.ToValidUTF8(string(b.head), ""), strings.ToValidUTF8(string(b.tail), "")
	return head + "\n[output truncated at 16 KiB; slice or filter in Python before printing]\n" + tail
}
