package gocli

import (
	"testing"
)

func TestBootRun(t *testing.T) {
	boot := CLI()
	msg := boot.Run([]string{"command", "help", "command", "-logf", "/Logs/gogen_new.log"})
	println(msg.Msg())
}
