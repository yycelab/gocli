package gocli

import (
	"fmt"
	"strconv"
	"strings"
)

type Flag interface {
	Name() string
	Alias() (string, bool)
	Usage() string
}
type FParser[T any] interface {
	Parse(args Args) (T, bool)
}

func NewFlag(nameUsageAlias ...string) Flag {
	size := len(nameUsageAlias)
	flag := flagOption[Args]{}
	if size > 0 {
		flag.name = nameUsageAlias[0]
	}
	if size > 1 {
		flag.usage = nameUsageAlias[1]
	}
	if size > 2 {
		flag.alias = nameUsageAlias[2]
	}

	return &flag
}

func BuildFlag[T any](name string, alias string, parser FlagParser[T]) Flag {
	return &flagOption[T]{name: name, alias: alias, parser: parser}
}

type FlagParser[T any] func(args Args) (T, bool)

type flagOption[T any] struct {
	name   string
	alias  string
	parser func(args Args) (T, bool)
	usage  string
}

func (opt *flagOption[T]) Usage() string {
	return opt.usage
}

func (opt *flagOption[T]) Name() string {
	return fmt.Sprintf("-%s", opt.name)
}
func (opt *flagOption[T]) Alias() (string, bool) {
	if len(opt.alias) > 0 {
		return fmt.Sprintf("--%s", opt.alias), true
	}
	return "", false
}
func (opt *flagOption[T]) Parse(args Args) (T, bool) {
	return opt.parser(args)
}

type FlagMap interface {
	GetString(key string) (v string, found bool)
	GetStrings(key string) (v Args, found bool)
	GetBool(key string) (v bool, found bool)
	GetInt(key string) (v int, found bool)
	Set(key string, values ...string)
	HasFlag(flag Flag) (Args, bool)
	Empty() bool
	toArgs() Args
}

func ParseFlag[T any](fm FlagMap, f Flag) (T, bool) {
	args, ok := fm.HasFlag(f)
	var v T
	if ok {
		if p, canParse := f.(FParser[T]); canParse {
			v, _ = p.Parse(args)
		}
	}
	return v, ok
}

func NewFlagMap() FlagMap {
	return &flagArgs{args: make(map[string][]string, 3)}
}

func NewFMap(args []string) FlagMap {
	fmap := make(map[string][]string, len(args)/2)
	fargIndex := -1
	var flag string
	for i := range args {
		if strings.HasPrefix(args[i], "-") {
			if fargIndex > -1 {
				fmap[flag] = args[fargIndex+1 : i]
			}
			flag = args[i]
			fargIndex = i
		}
	}
	if fargIndex > -1 {
		if fargIndex+1 < len(args) {
			fmap[flag] = args[fargIndex+1:]
		} else {
			fmap[flag] = make([]string, 0)
		}
	}
	return &flagArgs{args: fmap}
}

func fmapToString(fmap FlagMap) string {
	var w strings.Builder
	fargs := fmap.(*flagArgs)
	if fargs.args == nil {
		w.WriteString("(empty flagmap)")
		return w.String()
	}
	for i := range fargs.args {
		v := fargs.args[i]
		w.WriteString("key:")
		w.WriteString(i)
		w.WriteString(",value:")
		if len(v) == 0 {
			w.WriteString("(empty);")
		} else if len(v) == 1 {
			w.WriteString(v[0])
		} else {
			w.WriteString(fmt.Sprintf("[%s]", strings.Join(v, ",")))
		}
	}

	return w.String()
}

type flagArgs struct {
	args map[string]Args
}

func (fa *flagArgs) toArgs() Args {
	args := make([]string, 0, len(fa.args)*2)
	for flag := range fa.args {
		values := fa.args[flag]
		args = append(args, flag)
		args = append(args, values...)
	}
	return args
}

func (fa *flagArgs) HasFlag(flag Flag) (Args, bool) {
	args, ok := fa.GetStrings(flag.Name())
	if !ok {
		alias, set := flag.Alias()
		if set {
			args, ok = fa.GetStrings(alias)
		}
	}
	return args, ok
}
func (fa *flagArgs) GetString(key string) (v string, found bool) {
	args, ok := fa.args[key]
	found = ok
	if len(args) > 0 {
		v = args[0]
	}
	return
}
func (fa *flagArgs) GetStrings(key string) (v Args, found bool) {
	v, found = fa.args[key]
	return
}
func (fa *flagArgs) GetBool(key string) (v bool, found bool) {
	args, ok := fa.args[key]
	found = ok
	if len(args) > 0 {
		v = strings.Contains("|1|t|T|Y|y|yes|YES|TRUE|true|", fmt.Sprintf("|%s|", args[0]))
	}
	return
}

func (fa *flagArgs) GetInt(key string) (v int, found bool) {
	args, ok := fa.args[key]
	found = ok
	if len(args) > 0 {
		i, _ := strconv.ParseInt(args[0], 10, 32)
		v = int(i)
	}
	return
}

func (fa *flagArgs) Set(key string, values ...string) {
	fa.args[key] = values
}

func (fa *flagArgs) Empty() bool {
	return len(fa.args) == 0
}
