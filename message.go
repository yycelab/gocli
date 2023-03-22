package gocli

import "fmt"

type Message interface {
	// code < 0 程序应该中断
	Code() int
	Kind() int
	Msg() string
	Err() (error, bool)
}

type message struct {
	code int
	kind int
	msg  string
}

func (msg *message) Error() string {
	t, ok := levelMap[msg.kind]
	if !ok {
		t = "未定义"
	}
	return fmt.Sprintf("errcode:%d,kind:%s,msg:%s", msg.code, t, msg.msg)
}

func (msg *message) Err() (error, bool) {
	if msg.code < 0 || msg.kind > LOG_WARN {
		return msg, true
	}
	return nil, false
}
func (msg *message) Code() int {
	return msg.code
}
func (msg *message) Kind() int {
	return msg.kind
}
func (msg *message) Msg() string {
	return msg.msg
}

func InteruptMessage(msg string) Message {
	return NewMessage(-1, msg, LOG_INFO)
}

var NotImplementsMessage = NewMessage(0, "该功能还没实现", LOG_WARN)

func NewMessage(code int, msg string, kind int, msgargs ...any) Message {
	fmtstr := msg
	if len(msgargs) > 0 {
		fmtstr = fmt.Sprintf(msg, msgargs...)
	}
	return &message{code: code, msg: fmtstr, kind: kind}
}

func ErrMessage(code int, msg string, msgargs ...any) Message {
	return NewMessage(code, msg, LOG_ERROR, msgargs...)
}

func InfoMessage(code int, msg string, msgargs ...any) Message {
	return NewMessage(code, msg, LOG_INFO, msgargs...)
}

func WarnMessage(code int, msg string, msgargs ...any) Message {
	return NewMessage(code, msg, LOG_WARN, msgargs...)
}

func DebugMessage(code int, msg string, msgargs ...any) Message {
	return NewMessage(code, msg, LOG_DEBUG, msgargs...)
}

func SuccMessage(code int, msg string, msgargs ...any) Message {
	return NewMessage(code, msg, LOG_SUCC, msgargs...)
}
