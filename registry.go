package gocli

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

type RootCommandVisitor = func(key string, roots map[string]*RegisteredCommand) (next bool)

type PluginVisitor = func(key string, plugins map[string]*RegisteredPlugin) (next bool)

type CommandVisitor = func(key string, command map[string]Command) (next bool)

type Registry interface {
	RootCommand(key string) (RegisteredCommand, bool)
	// root command key max len
	RootKeyMaxLen() (int, bool)
	//支持 root command ,和subcommand
	Command(args []string) (RegisteredCommand, bool)
	CommandKeyMaxLen() (int, bool)
	Plugin(name string) (RegisteredPlugin, bool)
	FindPlugin(plugin Plugin) (RegisteredPlugin, bool)
	RegisterCommand(p Plugin, cmds ...Command) (ok bool, err error)
	RegisterPlugins(plugins ...Plugin)
	RangeRootCommand(RootCommandVisitor)
	RangePlugin(PluginVisitor)
	Finish(panicunfinished bool) (loaded int, failed int)
	Logger(log Log)
}

func NewRegistry() Registry {
	return &registration{
		plugins: make(map[string]*RegisteredPlugin, 5),
		roots:   make(map[string]*RegisteredCommand, 13),
	}
}

type registration struct {
	plugins      map[string]*RegisteredPlugin
	roots        map[string]*RegisteredCommand
	commands     map[string]*RegisteredCommand
	rootsMaxL    int
	commandsMaxL int
	dofinish     bool
	log          Log
	// mux     sync.Mutex
}

func (r *registration) Logger(log Log) {
	r.log = log
}

func (r *registration) Command(args []string) (RegisteredCommand, bool) {
	keys := strings.Join(args, " ")
	var rc RegisteredCommand
	v, ok := r.commands[keys]
	if ok {
		rc = *v
	}
	return rc, ok
}

func (r *registration) RootKeyMaxLen() (int, bool) {
	return r.rootsMaxL, r.dofinish
}
func (r *registration) CommandKeyMaxLen() (int, bool) {
	return r.commandsMaxL, r.dofinish
}

func (r *registration) Finish(panicunfinished bool) (loaded int, failed int) {
	for _, rp := range r.plugins {
		if !rp.finish {
			if panicunfinished {
				panic(fmt.Sprintf("加载插件%s%s失败", rp.Name(), rp.Version()))
			}
			failed++
			continue
		}
		loaded++
	}
	// commands mapping
	commands := make(map[string]*RegisteredCommand, 13)
	r.RangeRootCommand(func(key string, roots map[string]*RegisteredCommand) (next bool) {
		root := roots[key]
		if root.Command != nil {
			commands[root.Key()] = root
		}
		if len(root.Key()) > r.rootsMaxL {
			r.rootsMaxL = len(root.Key())
		}
		root.Range(func(key string, subs map[string]*RegisteredCommand) (next bool) {
			sc := subs[key].Command
			commands[sc.Key()] = subs[key]
			if len(sc.Key()) > r.commandsMaxL {
				r.commandsMaxL = len(sc.Key())
			}
			return true
		})
		return true
	})
	r.commands = commands
	r.dofinish = true
	return
}

func (r *registration) FindPlugin(plugin Plugin) (RegisteredPlugin, bool) {
	return r.Plugin(plugin.Name())
}

func (r *registration) RootCommand(key string) (RegisteredCommand, bool) {
	var rc RegisteredCommand
	v, ok := r.roots[key]
	if ok {
		rc = *v
	}
	return rc, ok
}
func (r *registration) Plugin(name string) (RegisteredPlugin, bool) {
	var rp RegisteredPlugin
	v, ok := r.plugins[name]
	if ok {
		rp = *v
	}
	return rp, ok
}

func (r *registration) RegisterCommand(p Plugin, cmds ...Command) (ok bool, err error) {
	plugin, ok := r.plugins[p.Name()]
	logger := r.log
	if !ok {
		panic(fmt.Sprintf("RegisterCommand只能在插件%s注册之后调用", p.Name()))
	}
	for i := range cmds {
		cmd := cmds[i]
		keys := strings.Fields(cmd.Key())
		rootKey := keys[0]
		root, found := r.roots[rootKey]
		if !found {
			root = NewRootRegistry(plugin.ptr, rootKey)
			if len(keys) == 1 {
				root.RootCommand(cmd)
			}
			r.roots[rootKey] = root
			err := plugin.Append(root)
			logger.Debug("%sregister command %s,err:%+v", plugin.Name(), root.Key(), err)
		} else if len(keys) == 1 && root.From != plugin.ptr {
			err = fmt.Errorf("根指令%s已注册:%+v,请求:%+v", rootKey, root.From, plugin.ptr)
			logger.Debug(err.Error())
			return
		}
		if len(keys) > 1 {
			add, found, e := root.AddSub(plugin.ptr, cmd)
			if !found {
				err = e
				logger.Debug("%s register subs command [%s] failed err:%+v", cmd.Key(), plugin.Name(), err)
				return
			}
			if e == nil {
				err := plugin.Append(add)
				logger.Debug("%s register subs command [%s] success ,err:%+v", plugin.Name(), cmd.Key(), err)
			}
		}
	}
	ok = true
	return
}
func (r *registration) RegisterPlugins(plugins ...Plugin) {

	for i := range plugins {
		p := plugins[i]

		v, ok := r.plugins[p.Name()]
		if ok && p != v.Plugin {
			panic(fmt.Sprintf("插件注册:重复%s,已注册:%+v%s,请求:%+v%s", p.Name(), v, v.Version(), p, p.Version()))
		}

		rp := &RegisteredPlugin{
			Plugin: p,
			ptr:    unsafe.Pointer(&p),
		}
		lp, ok := p.(*LoadedPlugin)
		if ok {
			rp.file = lp.File
			rp.sign = &signature{kind: algorithm_md5, digest: lp.Digest, verified: lp.Verified}
		}
		r.plugins[p.Name()] = rp
		r.log.Debug("注册插件%s", p.Name())
		ok, _ = r.RegisterCommand(rp, p.Registry()...)
		if !ok && !r.removeCommand(rp) {
			panic("插件注册:注册指令失败")
		}
	}
}

func (r *registration) removeCommand(p *RegisteredPlugin) bool {
	nboot := make(map[string]*RegisteredCommand, len(r.roots))
	ptr := p.ptr
	for k, rc := range r.roots {
		subs, ok := rc.HasSub()
		if rc.From == ptr && !ok {
			continue
		}
		nsub := make(map[string]*RegisteredCommand, len(subs))
		for sk, src := range subs {
			if src.From == ptr {
				continue
			}
			nsub[sk] = src
		}
		if rc.From == ptr && len(nsub) > 0 {
			return false
		}
		rc.SubCommands = nsub
		nboot[k] = rc
	}
	return true
}

func (r *registration) RangeRootCommand(v RootCommandVisitor) {
	cmap := r.roots
	for k := range r.roots {
		if !v(k, cmap) {
			break
		}
	}
}
func (r *registration) RangePlugin(v PluginVisitor) {
	pmap := r.plugins
	for k := range r.plugins {
		if !v(k, pmap) {
			break
		}
	}
}

type RegisteredCommandVisitor = RootCommandVisitor

type RegisteredCommand struct {
	From        unsafe.Pointer
	SubCommands SubCommand
	// mlsc        int //子指令最大长度
	key  string
	once sync.Once
	Command
}

func (c *RegisteredCommand) Range(v RegisteredCommandVisitor) {
	subs := c.SubCommands
	for i := range c.SubCommands {
		if !v(i, subs) {
			break
		}
	}
}

func (c *RegisteredCommand) Children() (children []*RegisteredCommand, maxL int) {
	children = make([]*RegisteredCommand, 0, len(c.SubCommands))
	for k := range c.SubCommands {
		if len(k) > maxL {
			maxL = len(k)
		}
		children = append(children, c.SubCommands[k])
	}
	return
}

func (c *RegisteredCommand) Usage() string {
	if c.Command != nil {
		return c.Command.Usage()
	}
	return ""
}
func (c *RegisteredCommand) Run(ctx Context, args []string, flagmap FlagMap) Message {
	if c.Command != nil {
		return c.Command.Run(ctx, args, flagmap)
	}
	return nil
}
func (c *RegisteredCommand) Flags() []Flag {
	if c.Command != nil {
		return c.Command.Flags()
	}
	return make([]Flag, 0)
}

func (c *RegisteredCommand) Key() string {
	return c.key
}

func (c *RegisteredCommand) HasSub() (sub SubCommand, has bool) {
	has = c.SubCommands != nil && len(c.SubCommands) > 0
	if has {
		sub = c.SubCommands
	}
	return
}

func (c *RegisteredCommand) RootCommand(cmd Command) {
	c.Command = cmd
}

func (c *RegisteredCommand) SetSubs(subs ...*RegisteredCommand) {
	tmp := make(map[string]*RegisteredCommand, len(subs))
	for i := range subs {
		tmp[subs[i].Key()] = subs[i]
	}
	c.SubCommands = tmp
}

func (c *RegisteredCommand) AppendSub(sub *RegisteredCommand) (ok bool, err error) {
	if sub.Command == nil {
		err = errors.New("子指令不可为空")
		return
	}
	keys := strings.Fields(sub.Command.Key())
	if len(keys) != 2 {
		err = errors.New("命令只能两层")
		return
	}
	p := keys[0]
	if p != c.key {
		err = fmt.Errorf("不是%s的子命令", c.key)
		return
	}
	subkey := keys[1]
	v, found := c.SubCommands[subkey]
	if found && v.Command != sub.Command {
		err = fmt.Errorf("%s已经存在%+v", subkey, v)
		return
	}
	// if len(subkey) > c.mlsc {
	// 	c.mlsc = len(subkey)
	// }
	c.SubCommands[subkey] = sub
	ok = true
	return
}

func (c *RegisteredCommand) AddSub(ptr unsafe.Pointer, cmd Command) (add *RegisteredCommand, ok bool, err error) {
	if cmd == nil {
		err = errors.New("子指令不可为空")
		return
	}
	keys := strings.Fields(cmd.Key())
	if len(keys) != 2 {
		err = errors.New("命令只能两层")
		return
	}
	p := keys[0]
	if p != c.key {
		err = fmt.Errorf("不是%s的子命令", c.key)
		return
	}
	subKey := keys[1]
	c.once.Do(func() {
		if c.SubCommands == nil {
			c.SubCommands = make(map[string]*RegisteredCommand, 5)
		}
	})
	v, ok := c.SubCommands[subKey]
	if ok {
		err = fmt.Errorf("%s已经存在%+v", subKey, v)
		return
	}
	// if len(subKey) > c.mlsc {
	// 	c.mlsc = len(subKey)
	// }
	add = &RegisteredCommand{
		Command: cmd,
		key:     subKey,
		From:    ptr,
	}
	c.SubCommands[subKey] = add
	ok = true
	return
}

type SubCommand = map[string]*RegisteredCommand

func NewRegistryCommand(ptr unsafe.Pointer, c Command) *RegisteredCommand {
	rc := &RegisteredCommand{
		From: ptr,
	}
	keys := strings.Fields(c.Key())
	if len(keys) > 0 {
		rc.key = keys[0]
	}
	if len(keys) > 1 {
		rc.AddSub(ptr, c)
	} else {
		rc.Command = c
	}
	return rc
}

func NewRootRegistry(from unsafe.Pointer, key string) *RegisteredCommand {
	return &RegisteredCommand{
		From:        from,
		key:         key,
		SubCommands: make(map[string]*RegisteredCommand, 5),
	}
}

type algorithm string

const (
	algorithm_md5 algorithm = "md5"
)

type signature struct {
	kind     algorithm
	digest   string
	verified bool
}

type RegisteredPlugin struct {
	Plugin
	sign   *signature
	finish bool
	ptr    unsafe.Pointer
	file   string
	root   map[string]*RegisteredCommand
	once   sync.Once
}

func (rp *RegisteredPlugin) init() {
	rp.once.Do(func() {
		if rp.root == nil {
			rp.root = make(map[string]*RegisteredCommand, 7)
		}
	})
}

func (rp *RegisteredPlugin) Commands() (commands []*RegisteredCommand, maxL int) {
	rp.init()
	commands = make([]*RegisteredCommand, 0, len(rp.root))
	for k, rc := range rp.root {
		if len(k) > maxL {
			maxL = len(k)
		}
		commands = append(commands, rc)
	}
	return
}

func (rp *RegisteredPlugin) Installed() {
	rp.finish = true
}

func (rp *RegisteredPlugin) Append(c *RegisteredCommand) error {
	rp.init()
	if rp.ptr != c.From {
		return fmt.Errorf("%s不能注册在%s下", c.Key(), rp.Name())
	}

	key := c.Key()
	if c.Command != nil {
		key = c.Command.Key()
	}
	keys := strings.Fields(key)
	v, ok := rp.root[keys[0]]
	if ok && v.From != c.From {
		return fmt.Errorf("该指令已经注册在别的插件下")
	}
	if !ok {
		v = NewRootRegistry(c.From, key)
		rp.root[keys[0]] = v
	}
	if len(keys) == 1 && c.Command != nil && v.Command == nil {
		v.Command = c.Command
	}
	if len(keys) > 1 {
		v.AppendSub(c)
	}
	return nil
}

func matchCommand(registry Registry, args []string) (c RegisteredCommand, cargs Args, matched bool) {
	if len(args) == 0 {
		return
	}
	c, matched = registry.RootCommand(args[0])
	if !matched {
		return
	}

	cargs = args[1:]
	if subs, has := c.HasSub(); has && len(args) > 1 {
		sc, ok := subs[args[1]]
		if ok {
			cargs = cargs[1:]
			c = *sc
			return
		}
	}
	if c.Command == nil {
		matched = false
		return
	}
	return
}
