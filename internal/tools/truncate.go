package tools

import (
	"fmt"
	"unicode/utf8"
)

const MaxToolOutputBytes = 16 * 1024 // 16KB حد أقصى لمخرج الأداة الواحدة

// TruncateOutput يقص النص الطويل مع إبقاء المقدمة والخاتمة دون كسر ترميز UTF-8.
func TruncateOutput(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	headLen := (limit * 2) / 3
	tailLen := limit - headLen

	head := validUTF8Prefix(s, headLen)
	tail := validUTF8Suffix(s, tailLen)
	omitted := len(s) - len(head) - len(tail)

	return fmt.Sprintf("%s\n\n... [truncated %d bytes] ...\n\n%s", head, omitted, tail)
}

func validUTF8Prefix(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}

func validUTF8Suffix(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	start := len(s) - maxBytes
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
}
