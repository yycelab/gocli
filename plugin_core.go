package gocli

import (
	"fmt"
	"sort"
	"strings"
)

// inspect plugin,command,subcommands
// quit
// generate plugin template file

const (
	core_name    = "gogen_core"
	core_usage   = "基础功能提供"
	core_version = "v0.0.1"
	twidth       = 140
	indent       = "  "
)

type historyItems = [][]string

var (
	pname    = NewFlag("nm")
	pversion = NewFlag("ver")
	pexport  = NewFlag("typ")
	pusage   = NewFlag("rmk")
)

var (
	_genplugin = NewCommand("genplugin", "生成一个插件模板文档 eg: genplugin plugin/demo/plugin.go -nm demo_plugin -ver ", BuildRun(
		func(ctx Context, args []string, flagmap FlagMap) Message {
			file := args[0]
			name, _ := flagmap.GetString(pname.Name())
			version, _ := flagmap.GetString(pversion.Name())
			usage, _ := flagmap.GetString(pusage.Name())
			export, _ := flagmap.GetString(pexport.Name())
			bean := &PluginBean{
				Name:       name,
				Version:    version,
				Usage:      usage,
				ExportType: export,
			}
			err := RenderPluginFile(bean, file)
			if err != nil {
				return ErrMessage(0, err.Error())
			}
			return InfoMessage(0, "生成pluginfile成功:%s", file)
		},
		InputRules(
			ExactlyLength(1, nil),
			MustFlagged(
				&FlagExpected{Flag: pname, Required: true, Validator: ExactlyLength(1, nil)},
				&FlagExpected{Flag: pversion, DefVal: "v0.0.1"},
				&FlagExpected{Flag: pusage, DefVal: "something todo."},
			),
		),
	))
	_quit = NewCommand("quit", "退出程序", func(ctx Context, args []string, flagmap FlagMap) Message {
		ctx.Interupt()
		return InteruptMessage("退出程序")
	})
	_workspace = NewCommand("ws", "显示当前工作空间状态相关信息", BuildRun(func(ctx Context, args []string, flagmap FlagMap) Message {
		var w strings.Builder
		path := ctx.WorkDir()
		w.WriteString(fmt.Sprintf("WorkDir: %s", path))
		return InfoMessage(0, w.String())
	}, InputRules(EmptyArgs())))
	_history = NewCommand("show history", "显示历史", func(ctx Context, args []string, flagmap FlagMap) Message {
		var w strings.Builder
		v := ctx.Value(history_list)
		if v != nil {
			items, ok := v.(*historyItems)
			if !ok {
				w.WriteString("history type error")
			} else {
				histories := *items
				for i := range histories {
					w.WriteString(fmt.Sprintf("%d. ", i+1))
					w.WriteString(strings.Join(histories[i], " "))
					w.WriteString("\n")
				}
			}
		}
		if w.Len() == 0 {
			w.WriteString("空")
		}
		return InfoMessage(0, w.String())
	})
	_plugins = NewCommand("show plugins", "查看加载的插件列表", func(ctx Context, args []string, flagmap FlagMap) Message {
		info := ctx.RegisteredPlugins()
		var w strings.Builder
		w.WriteString("gogen加载插件信息:\n")
		keys := make([]string, 0, len(info))
		max := 0
		for k := range info {
			keys = append(keys, k)
			if len(k) > max {
				max = len(k)
			}
		}
		sort.Strings(keys)
		for i := range keys {
			bundle := info[keys[i]]
			w.WriteString(fmt.Sprintf(
				"%s%s %s|md5(%s)|file(%s)\n",
				keys[i],
				strings.Repeat(" ", max-len(keys[i])),
				bundle.Version(),
				bundle.Md5(),
				bundle.File(),
			))
		}
		return InfoMessage(0, w.String())
	})
	_help = NewCommand("help", "使用方法,简述", BuildRun(func(ctx Context, args []string, flagmap FlagMap) Message {
		return InfoMessage(0, helpFunc(ctx))
	}, InputRules(ExactlyLength(0, ErrMessage(0, "不需要其它参数")))))
	_command     = NewRootCommand("command", "指令使用帮助信息")
	_commandHelp = NewCommand("command help", "指令使用说明", BuildRun(func(ctx Context, args []string, flagmap FlagMap) Message {
		return InfoMessage(0, commandHelp(ctx, args))
	}, InputRules(ExpectLength(1, 2, nil))))

	_commandSubs = NewCommand("command subs", "查看command下子指令用法", BuildRun(func(ctx Context, args []string, flagmap FlagMap) Message {
		return InfoMessage(0, subsHelp(ctx, args[0]))
	}, InputRules(ExactlyLength(1, nil))))

	_commandPlugin = NewCommand("command plugin", "查看plugin的下指令用法", BuildRun(func(ctx Context, args []string, flagmap FlagMap) Message {
		return InfoMessage(0, pluginHelp(ctx, args))
	}, InputRules(ExactlyLength(1, nil))))

	gogenCore = &GeneralPlugin{
		ID:   core_name,
		Desc: core_usage,
		Ver:  core_version,
		Commands: []Command{
			_quit,
			_plugins,
			_help,
			_command,
			_commandHelp,
			_commandSubs,
			_commandPlugin,
			_history,
			_workspace,
			_genplugin,
		},
	}
)

type RegistryCommands []*RegisteredCommand

func (g RegistryCommands) Len() int {
	return len(g)
}
func (g RegistryCommands) Less(i, j int) bool {
	f := g[i]
	s := g[j]
	fk := f.Key()
	if f.Command != nil {
		fk = f.Command.Key()
	}
	sk := s.Key()
	if s.Command != nil {
		sk = s.Command.Key()
	}
	if strings.Contains(fk, sk) {
		return true
	}
	if strings.Contains(sk, fk) {
		return false
	}
	return strings.Compare(fk, sk) < 1
}
func (g RegistryCommands) Swap(i, j int) {
	tmp := g[j]
	g[j] = g[i]
	g[i] = tmp
}

func pluginHelp(ctx Context, plugins []string) string {
	var w strings.Builder
	v := registrey(ctx)
	var (
		items RegistryCommands
		maxL  int
	)
	findstr := fmt.Sprintf("|%s|", strings.Join(plugins, "|"))
	indent := strings.Repeat(" ", 2)
	found := false

	v.RangePlugin(func(key string, plugins map[string]*RegisteredPlugin) (next bool) {
		if !strings.Contains(findstr, fmt.Sprintf("|%s|", key)) {
			return true
		}
		found = true
		rc := plugins[key]
		if rc.Helper() != nil {
			w.WriteString(rc.Helper()())
			return false
		}
		aw := &AlignWriter{}
		items, maxL = rc.Commands()
		if len(items) == 0 {
			aw.NopaddingAppend(fmt.Sprintf("插件%s注册指令: (空)\n", key))
			return true
		}
		sort.Sort(items)
		aw.NopaddingAppend(fmt.Sprintf("插件%s注册指令列表:\n", key))
		subTips := "{subcmd}"
		maxSubL := len(subTips)
		for i := range items {
			cr := items[i]
			usage := cr.Usage()
			if len(usage) == 0 {
				usage = fmt.Sprintf(`使用:"command subs %s"获取更多信息`, cr.Key())
			}

			subItems, subL := cr.Children()
			tips := subTips
			if len(subItems) == 0 {
				tips = ""
			}
			aw.RightPaddingAppend(
				fmt.Sprintf("%s%s%s %s", indent, cr.Key(), strings.Repeat(" ", maxL-len(cr.Key())), tips))
			aw.LeftPaddingAppend(usage).SplitLine(-1, false)
			flags := cr.Flags()
			if len(flags) > 0 {
				aw.LeftPaddingAppend("[flags options]").NewLine()
			}
			for i := range flags {
				f := flags[i]
				if a, ok := f.Alias(); ok {
					aw.LeftPaddingAppend(fmt.Sprintf("%s,%s %s", f.Name(), a, f.Usage())).SplitLine(-1, true)
				} else {
					aw.LeftPaddingAppend(fmt.Sprintf("%s %s", f.Name(), f.Usage())).SplitLine(-1, true)
				}
			}
			if subL > maxSubL {
				maxSubL = subL
			}
			var groups RegistryCommands = subItems
			sort.Sort(groups)
			for s := range groups {
				sub := subItems[s]
				aw.RightPaddingAppend(sub.Key()).Indent(maxL + 3)
				aw.LeftPaddingAppend(sub.Usage()).SplitLine(-1, false)
				flags := sub.Flags()
				if len(flags) > 0 {
					aw.LeftPaddingAppend("[flags options]").NewLine()
				}
				for i := range flags {
					f := flags[i]
					if a, ok := f.Alias(); ok {
						aw.LeftPaddingAppend(fmt.Sprintf("%s,%s %s", f.Name(), a, f.Usage())).SplitLine(-1, true)
					} else {
						aw.LeftPaddingAppend(fmt.Sprintf("%s %s", f.Name(), f.Usage())).SplitLine(-1, true)
					}
				}
			}
		}
		str := aw.MaskString(maxL+maxSubL+4, false)
		w.WriteString(str)
		return true
	})
	if !found {
		w.WriteString("无效的插件名称")
	}
	return w.String()
}

func subsHelp(ctx Context, name string) string {
	var w strings.Builder
	r := registrey(ctx)
	rc, ok := r.RootCommand(name)
	if !ok {
		w.WriteString(fmt.Sprintf("查无%s指令", name))
		return w.String()
	}
	items, maxL := rc.Children()
	if len(items) == 0 {
		w.WriteString("没有可用的子指令`")
	} else {
		w.WriteString(fmt.Sprintf("%s下子指令用法:\n", name))
		var children RegistryCommands = items
		sort.Sort(children)
		for i := range children {
			c := children[i]
			w.WriteString(fmt.Sprintf("%s%s %s",
				indent,
				fmt.Sprintf("%s%s", c.Key(), strings.Repeat(" ", maxL-len(c.Key()))),
				SplitLines(c.Usage(), twidth, maxL+3, false)))
		}
	}
	return w.String()
}

func commandHelp(ctx Context, args []string) string {
	var w strings.Builder
	r := registrey(ctx)
	find, ok := r.Command(args)
	if !ok {
		w.WriteString(fmt.Sprintf("查无此指令 %+v\n", args))
		return w.String()
	}
	w.WriteString(fmt.Sprintf("%s: %s", find.Key(), SplitLines(find.Usage(), twidth, len(find.Key())+1, false)))
	return w.String()
}

func registrey(ctx Context) Registry {
	return ctx.(*pluginContext).Context.(registreyContext).registry()
}

func helpFunc(ctx Context) string {
	var w strings.Builder
	w.WriteString(fmt.Sprintf("主程序:%s 版本:%s\n", core_name, core_version))
	w.WriteString("用法:Command [args...] [-Flag [flagArgs...] ...],可用指令:\n")
	r := registrey(ctx)
	max, _ := r.RootKeyMaxLen()
	var slice RegistryCommands = make([]*RegisteredCommand, 0, 5)
	r.RangeRootCommand(func(key string, roots map[string]*RegisteredCommand) (next bool) {
		rc := roots[key]
		slice = append(slice, rc)
		return true
	})
	sort.Sort(slice)
	for i := range slice {
		c := slice[i]
		w.WriteString(fmt.Sprintf("%s%s %s",
			indent,
			fmt.Sprintf("%s%s", c.Key(), strings.Repeat(" ", max-len(c.Key()))),
			SplitLines(c.Usage(), twidth, max+3, false)))
	}
	return w.String()
}
