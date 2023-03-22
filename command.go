package gocli

import "strings"

type Args = []string

type ExecFunc = func(ctx Context, args []string, flagmap FlagMap) Message

func NothingDo(ctx Context, args []string, flagmap FlagMap) (msg Message) {
	return InfoMessage(0, "请使用隐藏指令的子指令")
}

type Command interface {
	Key() string
	Usage() string
	Run(ctx Context, args []string, flagmap FlagMap) Message
	Flags() []Flag
}

func NewRootCommand(key string, usage string) Command {
	return NewCommand(key, usage, NothingDo)
}

func NewCommand(key string, usage string, do ExecFunc) Command {
	return &command{
		key:   key,
		usage: usage,
		run:   do,
	}
}
func NewFlagsCommand(key string, usage string, do ExecFunc, flags ...Flag) Command {
	return &command{
		key:   key,
		usage: usage,
		run:   do,
		flags: flags,
	}
}

type command struct {
	key   string
	usage string
	flags []Flag
	run   ExecFunc
}

func (c *command) Key() string {
	return c.key
}
func (c *command) Usage() string {
	return c.usage
}
func (c *command) Run(ctx Context, args []string, flagmap FlagMap) Message {
	return c.run(ctx, args, flagmap)
}

func (c *command) Flags() []Flag {
	return c.flags
}

func MergeFlagMap(args []string, fmap FlagMap) Args {
	if fmap == nil || fmap.Empty() {
		return args
	}
	return append(args, fmap.toArgs()...)
}

func ParseLine(input string) (Args, FlagMap) {
	return ParseInputArgs(strings.Fields(input))
}

func ParseInputArgs(args []string) (Args, FlagMap) {
	endPos := -1
	for i := range args {
		if strings.HasPrefix(args[i], "-") {
			endPos = i
			break
		}
	}
	if endPos > -1 {
		return args[0:endPos], NewFMap(args[endPos:])
	}
	return args, NewFMap(make([]string, 0))
}
