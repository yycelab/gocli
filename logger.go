package gocli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type RotateStrage = int

const (
	NONE RotateStrage = iota
	PARTIAL_NUMBER
	DATE_PATTERN
)

func NewLogger(file string) *Logger {
	log := &Logger{file: file, RStrage: NONE}
	pos := strings.LastIndex(file, ".")
	if pos > -1 {
		log.name = file[0:pos]
		log.suffix = file[pos:]
	}
	return log
}

type Prefix = func() string

// maxSize不低于2MB
func NewParialLogger(file string, maxMB int) *Logger {
	size := int64(1024 * 1024 * 2)
	maxSize := int64(maxMB * 1024 * 1024)
	if maxSize > size {
		size = maxSize
	}
	log := NewLogger(file)
	log.RStrage = PARTIAL_NUMBER
	log.Max = size
	return log
}

func NewDatePatternLogger(file string) *Logger {
	log := NewLogger(file)
	log.RStrage = DATE_PATTERN
	return log
}

type Logger struct {
	Max       int64
	RStrage   int
	nextcheck *time.Time
	file      string
	name      string
	suffix    string
	prefix    []any
}

func (log *Logger) PrependPrefix(values ...Prefix) {
	arr := make([]any, len(values))
	for i := range values {
		arr[i] = values[i]
	}
	log.prefix = append(log.prefix, arr...)
}

func (log *Logger) PrependPrefixString(values ...string) {
	arr := make([]any, len(values))
	for i := range values {
		arr[i] = values[i]
	}
	log.prefix = append(log.prefix, arr)
}

func (log *Logger) buildPrefix(w *bufio.Writer) int {
	num := 0
	for i := range log.prefix {
		item := log.prefix[i]
		switch t := item.(type) {
		case string:
			i, _ := w.WriteString(t)
			num += i
		case Prefix:
			i, _ := w.WriteString(t())
			num += i
		default:
			println(fmt.Sprintf("%+v", t))
		}
		w.WriteString(" ")
	}
	return num
}

func (log *Logger) NFile(f string) *Logger {
	lg := NewLogger(f)
	lg.RStrage = log.RStrage
	lg.Max = log.Max
	return lg
}

func (log *Logger) Write(p []byte) (n int, err error) {
	file, err := log.rotate()
	if err != nil {
		return 0, err
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	buff := bufio.NewWriter(f)
	num := log.buildPrefix(buff)
	n, err = buff.Write(p)
	buff.WriteString("\n")
	n += num + 1
	buff.Flush()
	return
}

func (log *Logger) rotate() (file string, err error) {
	switch log.RStrage {
	case DATE_PATTERN:
		now := time.Now()
		if log.nextcheck == nil || now.After(*log.nextcheck) {
			next := now.Add(time.Second * 5)
			log.nextcheck = &next
			dt := now.Format("20060102")
			log.file = fmt.Sprintf("%s_%s%s", log.name, dt, log.suffix)
		}
	case PARTIAL_NUMBER:
		f, err := os.Stat(log.file)
		if err == nil {
			if f.Size() >= log.Max {
				nfname := strings.TrimSuffix(log.file, log.suffix)
				pos := strings.LastIndex(nfname, "_")
				order := 1
				if pos > -1 {
					i, err := strconv.Atoi(nfname[order+1:])
					if err != nil {
						return "", err
					}
					order = i
				}
				log.file = fmt.Sprintf("%s_%d%s", log.name, order+1, log.suffix)
			}
		}
	default:
	}
	return log.file, nil
}

var DefaultDateFormatter = "20060102.15:04:05.999"

func fomattedNow(pattern string) string {
	pat := pattern
	if len(pattern) == 0 {
		pat = DefaultDateFormatter
	}
	return time.Now().Format(pat)
}
