package gocli

import (
	"path/filepath"
	"sync"
	"sync/atomic"
)

type ContextKey = string

const (
	history_list ContextKey = "history_*[][]string.registery"
	// interupt_signal   ContextKey = "exit.signal"
	// logfile_path      ContextKey = "logfile.registery"
	// console_bound     ContextKey = "console.registery"
	// basedata_registry ContextKey = "{plugins&commands}.registery"
	// work_dir          ContextKey = "workdir.current"
)

// key plugin name
// value plugin version
type PluginVersionMap = map[string]PluginBundle

type PluginBundle interface {
	Version() string
	Name() string
	Md5() string
	File() string
	Dependencies() []PluginBundle
}

type pluginBundle struct {
	*RegisteredPlugin
}

func (bundle *pluginBundle) Md5() string {
	md5 := "N/A"
	if bundle.RegisteredPlugin.sign != nil {
		md5 = bundle.sign.digest
	}
	return md5
}
func (bundle *pluginBundle) File() string {
	if len(bundle.file) == 0 {
		return "N/A"
	}
	return filepath.Base(bundle.file)
}
func (bundle *pluginBundle) Dependencies() []PluginBundle {
	return make([]PluginBundle, 0)
}

type Context interface {
	Value(key any) any
	Interupt() bool
	WorkDir() string
	RegisteredPlugins() PluginVersionMap
	StdConsole() StandConsole
	Logger() (logger Log, enable bool)
	ValueOperator
}

type registreyContext interface {
	registry() Registry
	Context
}

type ValueOperator interface {
	SetValueIfAbsent(key any, value any) (exist bool)
	SetValue(key any, value any)
	Remove(key any) (removed any, loaded bool)
}

type context struct {
	values    sync.Map
	registrey Registry
	console   Console
	workdir   string
	interrupt *atomic.Bool
}

func (ctx *context) WorkDir() string {
	return ctx.workdir
}

func (ctx *context) registry() Registry {
	return ctx.registrey
}

func (ctx *context) StdConsole() StandConsole {
	return ctx.console
}

func (ctx *context) Logger() (logger Log, enable bool) {
	if ctx.console == nil {
		return nil, false
	}
	return ctx.console.Log()
}

func (ctx *context) Remove(key any) (removed any, loaded bool) {
	return ctx.values.LoadAndDelete(key)
}

func (ctx *context) SetValue(key any, value any) {
	ctx.values.Store(key, value)
}

func (ctx *context) SetValueIfAbsent(key any, value any) bool {
	_, ok := ctx.values.LoadOrStore(key, value)
	return ok
}

func (ctx *context) Interupt() bool {
	old := ctx.interrupt.Load()
	return ctx.interrupt.CompareAndSwap(old, true)
}
func (ctx *context) Value(key any) any {
	v, _ := ctx.values.Load(key)
	return v
}

func (ctx *context) Console() Console {
	return ctx.console
}

func (ctx *context) RegisteredPlugins() PluginVersionMap {
	r := ctx.registrey
	vermap := make(map[string]PluginBundle, 7)
	if r == nil {
		return vermap
	}
	r.RangePlugin(func(key string, plugins map[string]*RegisteredPlugin) (next bool) {
		p := plugins[key]
		vermap[p.Name()] = &pluginBundle{
			RegisteredPlugin: p,
		}
		return true
	})
	return vermap
}
