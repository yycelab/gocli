package gocli

import (
	"fmt"
)

type Validator[T any] interface {
	Valid(ctx Context, value T) (bool, Message)
}

type ParamValidator[T any] interface {
	// 返回值小于0,实用所有参数
	Index() int
	Validator[T]
}

type ParamValidRule = func(ctx Context, val string) (bool, Message)

type ParamValiator = ParamValidator[string]

type ParamRule struct {
	index int // 小于0对所有参数成效
	valid ParamValidRule
	fail  Message
}

func ValidAllParam(rule ParamValidRule) ParamValiator {
	return &ParamRule{
		index: -1,
		valid: rule,
	}
}

func ValidParam(index uint, rule ParamValidRule) ParamValiator {
	return &ParamRule{
		index: int(index),
		valid: rule,
	}
}
func (vp *ParamRule) Index() int {
	return vp.index
}
func (vp *ParamRule) Valid(ctx Context, value string) (bool, Message) {
	ok, msg := vp.valid(ctx, value)
	return ok, FirstNoneNilResult(msg, vp.fail)
}

type CommandInputs struct {
	Args  Args
	Flags FlagMap
}

type InputValidator = Validator[*CommandInputs]

type inputValidRule struct {
	rule InputValidRule
	fail Message
}

func NewInputValidRule(rule InputValidRule, fail Message) InputValidator {
	return &inputValidRule{rule: rule, fail: fail}
}

func (iv *inputValidRule) Valid(ctx Context, value *CommandInputs) (bool, Message) {
	if value == nil {
		return false, ErrMessage(0, "无效的输入")
	}
	ok, msg := iv.rule(ctx, value.Args, value.Flags)
	return ok, FirstNoneNilResult(msg, iv.fail)
}

type InputValidRule = func(ctx Context, args []string, flags FlagMap) (bool, Message)

func EmptyArgs() Validator[*CommandInputs] {
	return NewInputValidRule(func(ctx Context, args []string, flags FlagMap) (bool, Message) {
		if len(args) > 0 {
			return false, ErrMessage(0, "该指令不带参数")
		}
		return true, nil
	}, nil)
}

func ExactlyLength(size uint, emsg Message) Validator[*CommandInputs] {
	valid := func(ctx Context, args []string, flags FlagMap) (pass bool, msg Message) {
		pass = len(args) == int(size)
		if !pass {
			msg = emsg
			if msg == nil {
				msg = ErrMessage(0, fmt.Sprintf("参数个数不匹配,期望:%d,输入:%d", size, len(args)))
			}
		}
		return
	}
	return NewInputValidRule(valid, nil)
}

func ExpectLength(min int, max int, emsg Message) Validator[*CommandInputs] {
	valid := func(ctx Context, args []string, flags FlagMap) (bool, Message) {
		inlen := len(args)
		if min > 0 && inlen < min {
			if emsg != nil {
				return false, emsg
			}
			return false, ErrMessage(0, fmt.Sprintf("参数个数不匹配:至少%d,输入:%d", min, inlen))
		}
		if max > 0 && inlen > max {
			if emsg != nil {
				return false, emsg
			}
			return false, ErrMessage(0, fmt.Sprintf("参数个数不匹配:至多%d,输入:%d", max, inlen))
		}
		return true, nil
	}
	return NewInputValidRule(valid, nil)
}

type FlagInputs = CommandInputs

type FlagValidator = Validator[*FlagInputs]

type FlagExpected struct {
	Flag      Flag
	Required  bool
	Validator FlagValidator
	ErrMsg    Message
	DefVal    any
}

func MustFlagged(flags ...*FlagExpected) Validator[*CommandInputs] {
	vaid := func(ctx Context, args []string, fmap FlagMap) (bool, Message) {
		for i := range flags {
			expected := flags[i]
			f, ok := fmap.HasFlag(expected.Flag)
			emsg := expected.ErrMsg
			if expected.Required && !ok {
				if emsg == nil {
					emsg = ErrMessage(0, fmt.Sprintf("Flag缺失,需要:%s", expected.Flag.Name()))
				}
				return false, emsg
			}
			if expected.DefVal != nil && len(f) == 0 {
				fmap.Set(expected.Flag.Name(), fmt.Sprintf("%+v", expected.DefVal))
			}
			if ok && expected.Validator != nil {
				if ok, msg := expected.Validator.Valid(ctx, &CommandInputs{Args: f, Flags: fmap}); !ok {
					emsg = msg
					if emsg == nil {
						emsg = ErrMessage(0, fmt.Sprintf("Flag%s不符合要求", expected.Flag.Name()))
					}
					return false, emsg
				}
			}
		}
		return true, nil
	}
	return NewInputValidRule(vaid, nil)
}

func InputRules(rules ...InputValidator) []InputValidator {
	return rules
}

func ParamRules(rules ...ParamValiator) []ParamValiator {
	return rules
}

func BuildRun(exec ExecFunc, inputRules []InputValidator, paramRules ...ParamValiator) ExecFunc {
	return func(ctx Context, args []string, flagmap FlagMap) Message {
		input := &CommandInputs{Args: args, Flags: flagmap}
		for i := range inputRules {
			ok, msg := inputRules[i].Valid(ctx, input)
			if !ok {
				return msg
			}
		}
		max := len(args)
		for i := range paramRules {
			r := paramRules[i]
			pindex := r.Index()
			if pindex < 0 {
				for j := range args {
					ok, msg := r.Valid(ctx, args[j])
					if !ok {
						return msg
					}
				}
				continue
			}
			if pindex < max {
				ok, msg := r.Valid(ctx, args[pindex])
				if !ok {
					return msg
				}
			}
		}
		return exec(ctx, args, flagmap)
	}
}

func FirstNoneNilResult[T any](values ...T) T {
	var r T
	for i := range values {
		var tmp any = values[i]
		if tmp != nil {
			return values[i]
		}
	}
	return r
}
