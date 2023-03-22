package gocli

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"unsafe"
)

type PluginContext interface {
	Context
	access(p unsafe.Pointer) bool
	accessPlugin(p Plugin) bool
}

type pluginContext struct {
	Context
	ptr unsafe.Pointer
}

func (pc *pluginContext) accessPlugin(p Plugin) bool {
	r := pc.Context.(registreyContext).registry()
	if v, ok := r.FindPlugin(p); ok {
		return v.ptr == pc.ptr
	}
	return false
}
func (pc *pluginContext) access(p unsafe.Pointer) bool {
	return pc.ptr == p
}
func NewPluginContext(ctx Context, plugin Plugin) PluginContext {
	var ptr unsafe.Pointer
	r := ctx.(registreyContext).registry()
	if v, ok := r.FindPlugin(plugin); ok {
		ptr = v.ptr
	}
	return &pluginContext{ptr: ptr, Context: ctx}
}

func NewPContext(ctx Context, ptr unsafe.Pointer) PluginContext {
	return &pluginContext{ptr: ptr, Context: ctx}
}

func MockContext() Context {
	return &context{}
}

type HelperFunc = func() string

type Plugin interface {
	Name() string
	Usage() string
	Version() string
	Setup(ctx Context) error
	//boot run 前检查
	BeforeRun(ctx Context) error
	Registry() []Command
	//all custom plugin's commands help info
	Helper() HelperFunc
}

type LifeHook = func(ctx Context) error

type GeneralPlugin struct {
	ID       string
	Desc     string
	Ver      string
	Init     LifeHook
	PreRun   LifeHook
	Commands []Command
	Help     HelperFunc
}

func (gp *GeneralPlugin) Helper() HelperFunc {
	return gp.Help
}

func (gp *GeneralPlugin) Name() string {
	if len(gp.ID) == 0 {
		return "unkown"
	}
	return gp.ID
}
func (gp *GeneralPlugin) Usage() string {
	return gp.Desc
}
func (gp *GeneralPlugin) Version() string {
	if len(gp.Ver) == 0 {
		return "unkown"
	}
	return gp.Ver
}
func (gp *GeneralPlugin) BeforeRun(ctx Context) error {
	if gp.PreRun == nil {
		return nil
	}
	return gp.PreRun(ctx)
}
func (gp *GeneralPlugin) Setup(ctx Context) error {
	if gp.Init == nil {
		return nil
	}
	return gp.Init(ctx)
}
func (gp *GeneralPlugin) Registry() []Command {
	if gp.Commands == nil {
		return make([]Command, 0)
	}
	return gp.Commands
}

const ExportPlugin = "Plugin"

type LoadedPlugin struct {
	File     string
	Digest   string
	Verified bool
	Plugin
}

func LoadPlugin(dirs []string, console Log) ([]*LoadedPlugin, int) {
	pluginFiles := make([]string, 0, 5)
	for i := range dirs {
		dir := dirs[i]
		p, err := os.Stat(dir)
		if err != nil || !p.IsDir() {
			console.Warn("[load plugin] scan dir:%s ,invalid dir", dir)
			continue
		}
		files, err := filepath.Glob(fmt.Sprintf("%s/*.so", dir))
		if err != nil {
			console.Warn("[load plugin] scan dir:%s ,invalid files", dir)
			continue
		}
		if len(files) > 0 {
			pluginFiles = append(pluginFiles, files...)
		}
	}
	num := len(pluginFiles)
	console.Info("[load plugin] found %d 个插件", num)
	verified := make([]*LoadedPlugin, 0, num)
	for i := range pluginFiles {
		pf := pluginFiles[i]
		pp, _ := filepath.Abs(pf)
		console.Info("preload plugin %s", pf)
		pn := filepath.Base(pp)
		p, err := plugin.Open(pp)
		if err != nil {
			console.Err("[load plugin] %s:open failed,%s", pn, err.Error())
			continue
		}
		plugin, err := p.Lookup(ExportPlugin)
		if err != nil {
			console.Err("[load plugin] %s:lookup failed", pn)
			continue
		}
		v, ok := plugin.(Plugin)
		if !ok {
			console.Err("[load plugin] %s:covert failed", pn)
			continue
		}
		console.Succ("[load plugin] Name:%s ,Version:%s", v.Name(), v.Version())
		var digest string
		{
			f, _ := os.Open(pp)
			hash := md5.New()
			io.Copy(hash, f)
			digest = hex.EncodeToString(hash.Sum(nil))
		}
		checkSign := false
		bts, err := os.ReadFile(fmt.Sprintf("%s.md5", pp))
		if err == nil {
			fileMd5 := string(bts)
			checkSign = fileMd5 == digest
		}
		verified = append(verified, &LoadedPlugin{Plugin: v, Digest: digest, Verified: checkSign, File: pf})
	}
	return verified, len(verified)
}
