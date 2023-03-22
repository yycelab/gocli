package gocli

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ShortCutKey = int

const (
	KeyAlt1 ShortCutKey = 161
	KeyAlt2 ShortCutKey = 8482
	KeyAlt3 ShortCutKey = 163
	KeyAlt4 ShortCutKey = 162
	KeyAlt5 ShortCutKey = 8734
	KeyAlt6 ShortCutKey = 167
	KeyAlt7 ShortCutKey = 182
	KeyAlt8 ShortCutKey = 8226
	KeyAlt9 ShortCutKey = 170
	KeyAlt0 ShortCutKey = 186
	KeyAltW ShortCutKey = 8721 // clean console
)

const (
	emptyHistoryTips = "(empty)"
	history_log      = "./history.log"
)

type ShortAction = int

const (
	ActionExec ShortAction = iota
	ActionInput
)

var shortcutTitle = map[ShortCutKey]string{
	KeyAlt1: "Alt+1",
	KeyAlt2: "Alt+2",
	KeyAlt3: "Alt+3",
	KeyAlt4: "Alt+4",
	KeyAlt5: "Alt+5",
	KeyAlt6: "Alt+6",
	KeyAlt7: "Alt+7",
	KeyAlt8: "Alt+8",
	KeyAlt9: "Alt+9",
	KeyAlt0: "Alt+0",
}

type ShortCommand struct {
	Command string
	Action  ShortAction
	*ShortKey
}

type ShortKey struct {
	Key   ShortCutKey
	Title string
}

func NewShortKey(key ShortCutKey) *ShortKey {
	v, found := shortcutTitle[key]
	if !found {
		v = ""
	}
	return &ShortKey{key, v}
}

func DefaultShortCut(custom ShortCutCommands) (ShortCutCommands, map[ShortCutKey]int) {
	shorts := ShortCutCommands{
		{ShortKey: NewShortKey(KeyAlt1), Command: "help", Action: ActionExec},
		{ShortKey: NewShortKey(KeyAlt2), Command: "show history", Action: ActionExec},
		{ShortKey: NewShortKey(KeyAlt3), Command: "show plugins", Action: ActionExec},
		{ShortKey: NewShortKey(KeyAlt4), Command: "command subs", Action: ActionInput},
		{ShortKey: NewShortKey(KeyAlt5), Command: "command plugin", Action: ActionInput},
	}
	smap := map[int]int{
		KeyAlt1: 0,
		KeyAlt2: 1,
		KeyAlt3: 2,
		KeyAlt4: 3,
		KeyAlt5: 4,
	}
	cshort := make(ShortCutCommands, 0, len(custom))
	index := 4
	for i := range custom {
		tmp := custom[i]
		if _, found := smap[custom[i].Key]; found {
			sk := NewShortKey(custom[i].Key * -1)
			sk.Title = strings.Repeat(" ", 5)
			tmp = ShortCommand{
				Command:  custom[i].Command,
				Action:   custom[i].Action,
				ShortKey: sk,
			}
		}
		cshort = append(cshort, tmp)
		smap[tmp.Key] = index + i
	}
	return append(shorts, cshort...), smap

}

func NewUi() Window {
	l, m := DefaultShortCut(nil)
	return &cliui{
		histories:  make([][]string, 0, 10),
		maxHistory: 10,
		goindex:    -1,
		shortcuts:  l,
		shortmap:   m,
	}
}

func ConfigUi(options UiOptions) Window {
	max := options.MaxHistoryItem
	if max < 3 {
		max = 3
	}
	l, m := DefaultShortCut(options.ShortCut)
	return &cliui{
		histories:  make([][]string, 0, max),
		maxHistory: max,
		goindex:    -1,
		shortcuts:  l,
		shortmap:   m,
	}
}

type ShortCutCommands = []ShortCommand

type UiOptions struct {
	MaxHistoryItem int
	ShortCut       ShortCutCommands
}

type Window interface {
	Run(prompt string, ctx Context) Message
}

type cliui struct {
	window     *tview.Application
	input      *tview.InputField
	hlist      *tview.Table
	console    *tview.TextView
	shortcuts  ShortCutCommands
	shortmap   map[ShortCutKey]int
	registry   Registry
	context    Context
	logger     Log
	histories  [][]string
	maxHistory int
	goindex    int
	// mux       sync.Mutex
}

func (ui *cliui) Run(prompt string, ctx Context) Message {
	ui.registry = ctx.(registreyContext).registry()
	log, ok := ctx.Logger()
	if !ok {
		return ErrMessage(-1, "获取logger失败")
	}
	ctx.SetValueIfAbsent(history_list, &ui.histories)
	ui.context = ctx
	ui.logger = log.NewLogger(history_log)
	app := tview.NewApplication()
	ui.window = app

	container := tview.NewFlex().SetFullScreen(false)
	container.SetBackgroundColor(tcell.ColorBlack)
	container.SetDirection(tview.FlexRow)
	container.AddItem(ui.helpView(), 0, 2, false)
	container.AddItem(ui.inputView(prompt), 0, 1, true)
	status := tview.NewFlex().SetDirection(tview.FlexColumn)
	status.AddItem(ui.shortcutView(), 0, 1, false)
	status.AddItem(ui.consoleView(), 0, 3, false)
	container.AddItem(status, 0, 17, false)
	ui.load()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if ui.dipatchInputKeyEvent(event) {
			return event
		}
		return nil
	})

	if err := app.SetRoot(container, true).Run(); err != nil {
		panic(err)
	}
	return InfoMessage(-1, "程序退出")
}

func (ui *cliui) load() {
	ui.command("help")
	// ui.focus(nil, ui.input)
}

func (ui *cliui) focus(old tview.Primitive, new tview.Primitive) {
	if old != nil {
		old.Blur()
	}
	ui.window.SetFocus(new)
}

func (ui *cliui) updateInput(txt string, from tview.Primitive) {
	ui.input.SetText(txt)
	if from != nil {
		ui.focus(from, ui.input)
	}
}

func (ui *cliui) historyGo(down bool) {
	if len(ui.histories) == 0 {
		return
	}

	defer func() {
		if ui.goindex > -1 && ui.goindex < len(ui.histories) {
			args := ui.histories[ui.goindex]
			ui.updateInput(fmt.Sprintf("%s ", strings.Join(args, " ")), nil)
		}
	}()
	if ui.goindex == -1 {
		if !down {
			ui.goindex = len(ui.histories) - 1
		}
		return
	}
	if down && ui.goindex+1 < len(ui.histories) {
		ui.goindex++
	}
	if !down && ui.goindex > 1 {
		ui.goindex--
	}
}

func (ui *cliui) isShortCut(e *tcell.EventKey) bool {
	key := e.Key()
	if key == 256 {
		code := int(e.Rune())
		if code == KeyAltW {
			ui.console.Clear()
			return true
		}
		if v, found := ui.shortmap[code]; found {
			c := ui.shortcuts[v]
			if c.Action == ActionExec {
				ui.command(c.Command)
			} else {
				ui.updateInput(fmt.Sprintf("%s ", c.Command), nil)
			}
			return true
		}
	}
	return false
}

func (ui *cliui) exit() {
	ui.appendConsole("程序3秒后退出!")
	go func() {
		<-time.After(time.Second * 3)
		ui.window.Stop()
	}()
}

func (ui *cliui) appendHistory(item string) {
	args := strings.Fields(item)
	if len(strings.Fields(item)) == 0 {
		return
	}
	items := ui.histories
	for i := range items {
		it := items[i]
		if strings.Join(it, "") == strings.Join(args, "") {
			return
		}
	}
	hlen := len(items)
	if hlen >= ui.maxHistory {
		copy(items, items[1:ui.maxHistory-1])
		items[ui.maxHistory-1] = args
	} else {
		ui.histories = append(items, args)
	}
}

func (ui *cliui) command(input string) {
	timestr := fomattedNow(DefaultDateFormatter)
	args, fmap := ParseInputArgs(strings.Fields(input))
	c, cargs, found := matchCommand(ui.registry, args)
	if !found {
		ui.appendConsole(fmt.Sprintf("%s>: %s\n%s", timestr, input, "没有匹配的指令\n"))
		return
	}

	message := c.Run(&pluginContext{
		Context: ui.context,
		ptr:     c.From,
	}, cargs, fmap)
	if message == nil {
		ui.appendConsole(fmt.Sprintf("%s>: %s\n%s", timestr, input, "返回空值\n"))
		ui.logger.Debug("command %s run return empty", c.Command.Key())
		return
	}
	if message.Code() < 0 {
		ui.exit()
		return
	}

	err, ok := message.Err()
	var msg string

	if ok {
		msg = err.Error()
	} else {
		msg = message.Msg()
	}
	print := fmt.Sprintf("%s>: %s\n%s", timestr, input, msg)
	ui.appendConsole(print)
	ui.logger.Debug("command %s run result ", c.Command.Key(), print)
}

func (ui *cliui) submit() {
	txt := strings.TrimSpace(ui.input.GetText())
	if len(txt) == 0 {
		return
	}
	ui.logger.Debug("submit %s", txt)
	ui.appendHistory(txt)
	ui.command(txt)
	ui.updateInput("", nil)
}

func (ui *cliui) appendConsole(msg string) {
	ui.console.Write([]byte(fmt.Sprintf("%s\n", msg)))
}

func (ui *cliui) consoleView() *tview.TextView {
	output := tview.NewTextView()
	ui.console = output
	output.SetMaxLines(40)
	output.SetBorder(true)
	output.SetTitle(" 信息")
	return output
}

func (ui *cliui) helpView() *tview.Table {
	help := tview.NewTable()
	tips := []string{"BLANK:空格提示", "ESC:清空输入", "Ctrl+C:退出程序", "Alt+W:清空信息"}
	var cell *tview.TableCell
	for i := range tips {
		cell = tview.NewTableCell(tips[i])
		cell.SetTextColor(tcell.ColorDarkOliveGreen)
		cell.SetMaxWidth(80)
		help.SetCell(0, i, cell)
	}
	return help
}

func (ui *cliui) shortcutView() *tview.Table {
	shortcut := tview.NewTable()
	ui.hlist = shortcut
	shorts := ui.shortcuts
	for k := range shorts {
		c := shorts[k]
		code := k
		if code < 0 {
			code = 0
		}
		shortcut.SetCellSimple(k, 0, fmt.Sprintf(" [%s] %s", c.Title, c.Command))
	}
	shortcut.SetTitle(" 快捷键 ").SetBorder(true).SetBorderAttributes(tcell.AttrDim)
	return shortcut
}

func (ui *cliui) dipatchInputKeyEvent(e *tcell.EventKey) bool {
	key := e.Key()
	switch key {
	case tcell.KeyUp:
		ui.historyGo(false)
	case tcell.KeyDown:
		ui.historyGo(true)
	case tcell.KeyEscape:
		ui.updateInput("", nil)
	case tcell.KeyCtrlC:
		ui.exit()
	case tcell.KeyEnter:
		fallthrough
	case tcell.KeyLF:
		ui.submit()
	case 256:
		return !ui.isShortCut(e)
	default:
		if ui.goindex > -1 {
			ui.goindex = -1
		}
		return true
	}
	return false
}

func (ui *cliui) inputView(prompt string) *tview.InputField {
	input := tview.NewInputField()
	ui.input = input
	input.SetLabelColor(tcell.ColorTeal)
	input.SetRect(0, 10, 240, 20)
	input.SetLabel(prompt)
	input.SetFieldBackgroundColor(tcell.ColorBlack)
	return input
}

// func autoCompelete(registry Registry) func(currentText string) (entries []string) {

// 	return func(currentText string) (entries []string) {
// 		commands := make([]string, 0, 7)
// 		args, flagmap := ParseInputArgs(strings.Fields(currentText))
// 		if len(args) == 0 {
// 			registry.RangeRootCommand(func(key string, roots map[string]*RegisteredCommand) (next bool) {
// 				commands = append(commands, key)
// 				return true
// 			})
// 			return commands
// 		}
// 		c, ok := registry.RootCommand(args[0])
// 		if !ok {
// 			commands = append(commands, fmt.Sprintf("%s指令无效", args[0]))
// 			return commands
// 		}
// 		emptyFlag := flagmap.Empty()
// 		subs, ok := c.HasSub()
// 		if len(args) == 1 && ok && emptyFlag {
// 			c.Range(func(key string, roots map[string]*RegisteredCommand) (next bool) {
// 				commands = append(commands, key)
// 				return true
// 			})
// 			return commands
// 		}
// 		var flags []Flag
// 		if c.Command != nil {
// 			flags = c.Flags()
// 		}
// 		if len(args) > 1 {
// 			sub, ok := subs[args[1]]
// 			if ok {
// 				flags = sub.Flags()
// 			}
// 		}
// 		if len(flags) > 0 {
// 			for i := range flags {
// 				if _, ok := flagmap.HasFlag(flags[i]); !ok {
// 					commands = append(commands, flags[i].Name())
// 				}
// 			}
// 		}
// 		return commands
// 	}
// }
