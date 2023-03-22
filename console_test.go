package gocli

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestFiledBuilder(t *testing.T) {
	wf := NewAlignWriter()
	wf.NopaddingAppend("dfjladkfjasdfhadkjhfadfhajdhfa").NewLine()
	wf.LeftPaddingAppend("command ").Indent(4)
	wf.LeftPaddingAppend("hello132dhfdhfjadhfk;").SplitFixedWidthLine(20, -1, false)
	d, _ := os.OpenFile("./console_test.txt", os.O_CREATE|os.O_WRONLY, 0644)
	defer d.Close()
	f := bufio.NewWriter(d)
	f.WriteRune(rune(' '))
	f.WriteRune(rune(' '))
	f.WriteRune(rune(' '))
	f.WriteRune(rune(' '))
	f.WriteRune(rune('e'))
	f.WriteRune(rune('\n'))
	f.WriteString(strings.Repeat(" ", 4))
	f.WriteString("d\n")
	wf.MaskWirte(12, f)
	f.Flush()
}
