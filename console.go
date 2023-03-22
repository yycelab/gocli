package gocli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
)

type LogLevel = int

const (
	LOG_DEBUG LogLevel = iota
	LOG_INFO
	LOG_SUCC
	LOG_WARN
	LOG_ERROR
)

var levelMap = map[int]string{
	0: "debug",
	1: "info",
	2: "succ",
	3: "warn",
	4: "error",
}

type StandConsole interface {
	Err(emsg string, args ...any)
	WError(err error)
	Warn(msg string, args ...any)
	Info(msg string, args ...any)
	Succ(msg string, args ...any)
	Debug(msg string, args ...any)
	Msg(msg Message)
	Level(level LogLevel)
	Prefix(prefix string, args ...any) Console
	AppendPrefix(prefix string, args ...any) Console
	PrependPrefix(prefix string, args ...any) Console
}

type Log interface {
	NewLogger(file string) Log
	StandConsole
}

// 控制台打印
type Console interface {
	//文件logger,如过没有设置,则使用指定文件记录日志
	Log() (Log, bool)
	Std() StandConsole
	//只在终端输出
	StandConsole
}

func NewConsole(term io.Writer, logf *Logger) Console {
	c := &mixedConsole{t: term, log: logf}
	c.ll.Store(int32(LOG_DEBUG))
	return c
}

type mixedConsole struct {
	prefix string
	t      io.Writer
	log    *Logger
	ll     atomic.Int32
}

func (console *mixedConsole) NewLogger(file string) Log {
	var log *Logger
	if console.log != nil {
		log = NewLogger(file)
		log.RStrage = console.log.RStrage
		log.Max = console.log.Max
	} else {
		log = NewParialLogger(file, 30)
	}
	return &mixedConsole{prefix: console.prefix, ll: console.ll, log: log}
}

func (console *mixedConsole) Std() StandConsole {
	return console
}

func (console *mixedConsole) Log() (Log, bool) {
	if console.log == nil {
		return nil, false
	}
	return console, true
}

func (console *mixedConsole) StandConsole() (Console, error) {
	if console.t == nil {
		return nil, errors.New("无效的Console,没有终端")
	}
	return &mixedConsole{prefix: console.prefix, t: console.t, ll: console.ll}, nil
}
func (console *mixedConsole) FileLogger() (Console, error) {
	if console.log == nil {
		return nil, errors.New("无效的Console,没有文件logger,请使用NewFileLogger获取Console")
	}
	return &mixedConsole{prefix: console.prefix, log: console.log, ll: console.ll}, nil
}

func (console *mixedConsole) Prefix(prefix string, args ...any) Console {
	pre := fmt.Sprintf(prefix, args...)
	return &mixedConsole{prefix: pre}
}

func (console *mixedConsole) AppendPrefix(prefix string, args ...any) Console {
	pre := fmt.Sprintf(prefix, args...)
	if len(console.prefix) == 0 {
		return &mixedConsole{prefix: pre}
	}
	return &mixedConsole{prefix: fmt.Sprintf("%s %s", console.prefix, pre)}
}
func (console *mixedConsole) PrependPrefix(prefix string, args ...any) Console {
	pre := fmt.Sprintf(prefix, args...)
	if len(console.prefix) == 0 {
		return &mixedConsole{prefix: pre}
	}
	return &mixedConsole{prefix: fmt.Sprintf("%s %s", pre, console.prefix)}
}

func (console *mixedConsole) write(ll LogLevel, txt string, args ...any) {
	if ll >= int(console.ll.Load()) {
		msg := txt
		if len(args) > 0 {
			msg = fmt.Sprintf(msg, args...)
		}
		wtxt := []byte(fmt.Sprintf("%s %s", levelMap[ll], msg))
		if console.log != nil {
			console.log.Write(wtxt)
		}
		if console.t != nil {
			console.t.Write(wtxt)
			console.t.Write([]byte("\n"))
		}
	}
}

func (console *mixedConsole) Err(emsg string, args ...any) {
	console.write(LOG_ERROR, emsg, args...)
}
func (console *mixedConsole) WError(err error) {
	console.write(LOG_ERROR, err.Error())
}
func (console *mixedConsole) Warn(msg string, args ...any) {
	console.write(LOG_WARN, msg, args...)
}
func (console *mixedConsole) Info(msg string, args ...any) {
	console.write(LOG_INFO, msg, args...)
}
func (console *mixedConsole) Succ(msg string, args ...any) {
	console.write(LOG_SUCC, msg, args...)
}
func (console *mixedConsole) Debug(msg string, args ...any) {
	console.write(LOG_DEBUG, msg, args...)
}
func (console *mixedConsole) Msg(msg Message) {
	console.write(msg.Kind(), msg.Msg())
}
func (console *mixedConsole) Level(level LogLevel) {
	console.ll.Swap(int32(level))
}

type PaddingType = int

const (
	SPLIT_LINE_WIDTH             = 140
	INDENT_NOLIMIT               = -1
	NONE_PADDING     PaddingType = iota
	RIGHT_PADDING
	LEFT_PADDING
)

type AlignField struct {
	Content     string
	AlignWidth  int
	Padding     PaddingType
	splitLine   bool
	firstIndent bool
	indent      int
	endln       bool
}

func writeString(writer io.Writer, txt string) {
	writer.Write([]byte(txt))
}

func (field *AlignField) WriteAppend(w io.Writer) {
	if field.splitLine {
		str := SplitLines(field.Content, field.AlignWidth, field.indent, field.firstIndent)
		writeString(w, str)
	} else {
		if field.indent > 0 {
			writeString(w, strings.Repeat(" ", field.indent))
		}
		content := []rune(field.Content)
		pad := field.AlignWidth - field.indent - len(content)
		if pad > 0 && field.Padding != NONE_PADDING {
			switch field.Padding {
			case LEFT_PADDING:
				writeString(w, fmt.Sprintf("%s%s", strings.Repeat(" ", pad), field.Content))
			case RIGHT_PADDING:
				writeString(w, fmt.Sprintf("%s%s", field.Content, strings.Repeat(" ", pad)))
			}
		} else {
			writeString(w, field.Content)
		}
	}
	if field.endln {
		writeString(w, "\n")
	}
}

func (field *AlignField) Indent(indent int) *AlignField {
	field.indent = indent
	return field
}

func (field *AlignField) NewLine() *AlignField {
	field.endln = true
	return field
}

func (field *AlignField) SplitFixedWidthLine(width int, indent int, firstIndent bool) *AlignField {
	field.Padding = LEFT_PADDING
	field.indent = indent
	field.splitLine = true
	field.AlignWidth = width
	field.firstIndent = firstIndent
	return field
}

func (field *AlignField) SplitLine(indent int, firstIndent bool) *AlignField {
	return field.SplitFixedWidthLine(SPLIT_LINE_WIDTH, indent, firstIndent)
}

func NewAlignWriter() *AlignWriter {
	return ScreenAlignWriter(SPLIT_LINE_WIDTH)
}

func ScreenAlignWriter(screenWidth int) *AlignWriter {
	return &AlignWriter{width: screenWidth}
}

type AlignWriter struct {
	fields []*AlignField
	width  int
}

func (aw *AlignWriter) NopaddingAppend(context string) *AlignField {
	return aw.AlignAppend(context, 0, NONE_PADDING)
}

func (aw *AlignWriter) RightPaddingAppend(context string) *AlignField {
	return aw.AlignAppend(context, INDENT_NOLIMIT, RIGHT_PADDING)
}

func (aw *AlignWriter) LeftPaddingAppend(context string) *AlignField {
	return aw.AlignAppend(context, INDENT_NOLIMIT, LEFT_PADDING)
}

func (aw *AlignWriter) AlignAppend(content string, width int, pad PaddingType) *AlignField {
	if aw.fields == nil {
		aw.fields = make([]*AlignField, 0, 10)
	}
	field := &AlignField{Content: content, Padding: pad, AlignWidth: width}
	aw.fields = append(aw.fields, field)
	return field
}

func (aw *AlignWriter) MaskWirte(align int, writer io.Writer) {
	for i := range aw.fields {
		f := aw.fields[i]
		if f.splitLine {
			if f.indent < 0 {
				f.indent = align
			}
		} else if f.AlignWidth < 0 || f.Padding == NONE_PADDING {
			f.AlignWidth = align
		}
		f.WriteAppend(writer)
	}
}

func (aw *AlignWriter) MaskString(align int, endln bool) string {
	var w strings.Builder
	aw.MaskWirte(align, &w)
	return w.String()
}
