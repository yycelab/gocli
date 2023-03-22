package gocli

import (
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const (
	boot_log = "/tmp/gogen_boot.go"
)

var (
	LogFlag   = NewFlag("logf", "-logf ./app.log 指定日志文件")
	UiFlag    = NewFlag("ui", "程序启用GUI,输入指令会忽略")
	PluginDir = NewFlag("pdir", "-pdir {dir} 添加插件目录,可多个")
	LogFLevel = NewFlag("logl", "-logl {0-5} 日志输出级别(0-5)对应 debug-error")
	WorkDir   = NewFlag("wdir", "-wdir 指定工作目录")
	CheckSum  = NewFlag("check", "-check") //验证插件签名
)

func CLI() *BootStrap {
	return &BootStrap{}
}

type BootStrap struct {
	stop atomic.Bool
}

func (boot *BootStrap) Run(args []string) Message {

	arg, fmap := ParseInputArgs(args)
	// run mode
	_, ok := fmap.HasFlag(UiFlag)
	context := boot.initContext(ok, arg, fmap)

	values, _ := fmap.HasFlag(PluginDir)
	registrey := context.registry()
	_, verify := fmap.HasFlag(CheckSum)
	boot.registerPlugin(context, verify, values)
	registrey.Finish(true)
	if ok {
		return NewUi().Run("> ", context)
	}
	return boot.exec(context, arg, fmap)
}

func (boot *BootStrap) initContext(ui bool, args []string, fmap FlagMap) registreyContext {
	if len(args) == 0 {
		fmap.Set(UiFlag.Name())
		pdir := "./plugins"
		f, err := os.Stat(pdir)
		if err == nil && f.IsDir() {
			fmap.Set(PluginDir.Name(), pdir)
		}
	}
	if v, ok := fmap.HasFlag(LogFlag); ok && len(v) == 0 {
		fmap.Set(LogFlag.Name(), "./gogen.log")
	}
	//context , console settings
	var logFile string
	values, ok := fmap.HasFlag(LogFlag)
	if ok && len(values) > 0 {
		logFile = values[0]
	}
	if len(logFile) == 0 {
		logFile = boot_log
	}

	w := NewParialLogger(logFile, 10)
	w.PrependPrefix(func() string {
		return time.Now().Format("2006-01-02 15:04:05.999")
	})
	var console Console
	if ui {
		console = NewConsole(nil, w)
	} else {
		console = NewConsole(os.Stdout, w)
	}
	registry := NewRegistry()
	log, _ := console.Log()
	registry.Logger(log)
	log.Info("设置logger,boot,registry")
	v, ok := fmap.GetInt(LogFLevel.Name())
	if !ok {
		v = LOG_INFO
	}
	console.Level(v)
	wdir, ok := fmap.GetString(WorkDir.Name())
	if !ok {
		wdir = "."
	}
	ctx := &context{
		console:   console,
		registrey: registry,
		interrupt: &boot.stop,
		workdir:   wdir,
	}

	//discover plugin
	log.Info("boot run ,args:[%s], flagmap:{%s}", strings.Join(args, ","), fmapToString(fmap))
	return ctx
}

func (boot *BootStrap) registerPlugin(ctx registreyContext, verify bool, pluginDir []string) error {
	boot.internalPluginRegister(ctx)
	logger, _ := ctx.Logger()
	register := ctx.registry()
	if len(pluginDir) > 0 {
		plugins, num := LoadPlugin(pluginDir, logger)
		logger.Info("found %d plugins", num)

		for _, lp := range plugins {
			if !verify || lp.Verified {
				logger.Info("install plugin %s,md5:%s,file:%s", lp.Name(), lp.Digest, lp.File)
				register.RegisterPlugins(lp)
			} else {
				logger.Err("Plugin name:%s,path:%s ;checksum failed", lp.Plugin.Name(), lp.File)
			}
		}

	}
	var err error
	cmap := make(map[string]PluginContext, 7)
	register.RangePlugin(func(key string, plugins map[string]*RegisteredPlugin) (next bool) {
		p := plugins[key]
		ctx := NewPluginContext(ctx, p.Plugin)
		if e := p.Setup(ctx); err != nil {
			err = e
			return false
		}
		cmap[key] = ctx
		return true
	})
	if err == nil {
		register.RangePlugin(func(key string, plugins map[string]*RegisteredPlugin) (next bool) {
			p := plugins[key]
			if e := p.BeforeRun(cmap[key]); e != nil {
				err = e
				return false
			}
			p.Installed()
			return true
		})
	}
	return err
}

func (boot *BootStrap) internalPluginRegister(context registreyContext) {
	context.registry().RegisterPlugins(gogenCore)
}

func (boot *BootStrap) exec(ctx registreyContext, args []string, fmap FlagMap) Message {
	if len(args) == 0 {
		return WarnMessage(0, "no command found")
	}
	c, arg, ok := matchCommand(ctx.registry(), args)
	if !ok || c.Command == nil {
		return WarnMessage(404, "command not found")
	}
	return c.Run(NewPContext(ctx, c.From), arg, fmap)
}
