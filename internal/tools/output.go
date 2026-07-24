package tools

import (
	"fmt"
	"unicode/utf8"
)

const toolOutputLimit = 50000

type limitedOutput struct {
	head  []byte
	tail  []byte
	total int
}

func (output *limitedOutput) Write(data []byte) (int, error) {
	written := len(data)
	output.total += written
	keepEach := toolOutputLimit / 2
	if len(output.head) < keepEach {
		keep := keepEach - len(output.head)
		if keep > len(data) {
			keep = len(data)
		}
		output.head = append(output.head, data[:keep]...)
		data = data[keep:]
	}
	if len(data) == 0 {
		return written, nil
	}
	combined := append(output.tail, data...)
	if len(combined) > keepEach {
		combined = combined[len(combined)-keepEach:]
	}
	output.tail = combined
	return written, nil
}

func (output *limitedOutput) String() string {
	if output.total <= toolOutputLimit {
		return string(append(output.head, output.tail...))
	}

	headLength := len(output.head)
	headRuneStart := headLength - 1
	for headRuneStart >= 0 && !utf8.RuneStart(output.head[headRuneStart]) {
		headRuneStart--
	}
	if headRuneStart >= 0 && !utf8.FullRune(output.head[headRuneStart:]) {
		headLength = headRuneStart
	}
	tailStart := 0
	for tailStart < len(output.tail) && !utf8.RuneStart(output.tail[tailStart]) {
		tailStart++
	}

	omitted := output.total - headLength - (len(output.tail) - tailStart)
	marker := fmt.Sprintf("\n\n[... truncated %d bytes ...]\n\n", omitted)
	return string(output.head[:headLength]) + marker + string(output.tail[tailStart:])
}
