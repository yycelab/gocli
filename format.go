package gocli

import (
	"bufio"
	"fmt"
	"strings"
	"unicode/utf8"
)

func SplitLines(txt string, width int, indent int, firstIndent bool) (fmtstr string) {
	wlen := utf8.RuneCountInString(txt)
	left := 0
	if firstIndent {
		left = indent
	}
	if wlen <= width && !strings.Contains(txt, "\n") {
		return fmt.Sprintf("%s%s\n", strings.Repeat(" ", left), txt)
	}
	r := bufio.NewScanner(strings.NewReader(txt))
	r.Split(bufio.ScanLines)
	sholdIndent := firstIndent
	var w strings.Builder
	var line string
	goto parse
parse:
	for r.Scan() {
		if !sholdIndent && w.Len() > 0 {
			sholdIndent = true
		}
		line = r.Text()
		goto wrap
	}
	fmtstr = w.String()
	return
wrap:
	chars := []rune(line)
	realWidth := width - indent
	l := len(chars)
	if l > realWidth {
		i := 0
		for {
			end := i + realWidth
			if i+realWidth > l {
				end = l
			}
			if sholdIndent || i > 0 {
				w.WriteString(strings.Repeat(" ", indent))
			}
			w.WriteString(string(chars[i:end]))
			w.WriteString("\n")
			if i == l {
				break
			}
			i = end
		}
	} else {
		if sholdIndent {
			w.WriteString(strings.Repeat(" ", indent))
		}
		w.WriteString(line)
		w.WriteString("\n")
	}
	goto parse
}
