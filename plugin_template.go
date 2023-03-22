package gocli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var tpl = `package main

import (
	"fmt"

	"yycelab.com/gocli"
)
// 确保引入 yycelab.com/gocli {版本同主程序一致},eg: go get yycelab.com/gocli@latest
// 插件的基本属性
const (
	name      = "{{.Name}}"
	version   = "{{.Version}}"
	usage     = "{{.Usage}}"
	{{- if gt (len .ExportType) 0}}
//	exportKey = "{{.Name}}.{{.ExportType}}.export" // {{len .ExportType}}
	{{end}}
)

{{- if gt (len .ExportType) 0}}
//自定义类型 ,其它类型看清况导入包
//type {{.ExportType}} interface{}
{{end}}

var (
// once       sync.Once  //一次初始化
{{- if gt (len .ExportType) 0}}
// export  {{.ExportType}}
{{end}}
)

// 插件初始化,对外暴露数据
func Setup(context gocli.Context) error {
	// 	once.Do(func() {
	// 		tmp := new({{.ExportType}}) //视情况完成初始化
	// 		if exist := context.SetValueIfAbsent(exportKey, &tmp); !exist {
	// 			export = tmp
	// 		}
	// 	})
	// 	if export == nil {
	// 		return errors.New("init {{.ExportType}} failed")
	// 	}
	return nil
}

// 已完成所有插件初始化完,运行环境检查,比如依赖处理等等
func BeforeRun(ctx gocli.Context) error {
	pluginMap := ctx.RegisteredPlugins()
	coreId := "gogen_core"
	if v, ok := pluginMap[coreId]; !ok || v.Md5() != "N/A" {
		return fmt.Errorf("未找到%s,version:%s,md5:%s", coreId, "v0.0.1", "N/A")
	}
	return nil
}

// declare commands
var (
	//空指令,根指令,指令使用方法秒速;一般用来打包功能块,下面有多个子指令
	cmd_root = gocli.NewRootCommand("{command}", "{指令使用方法}")
	//通过 BuildRun方法包裹 Command,实现参数的输入校验
	cmd_sub_withbuild = gocli.NewCommand(
		"{command} {subcommand}",
		"{指令用法描述}",
		gocli.BuildRun(
			func(ctx gocli.Context, args []string, flagmap gocli.FlagMap) gocli.Message {
				return gocli.NotImplementsMessage
			},
			gocli.InputRules(gocli.ExactlyLength(2, nil)),
		),
	)
	command_sub = gocli.NewCommand(
		"{command} {subcommand}",
		"{指令用法描述}",
		func(ctx gocli.Context, args []string, flagmap gocli.FlagMap) gocli.Message {
			return gocli.NotImplementsMessage
		},
	)
)

var Plugin = gocli.GeneralPlugin{
	ID:     name,
	Ver:    version,
	Desc:   usage,
	Init:   Setup,
	PreRun: BeforeRun,
	Commands: []gocli.Command{
		cmd_root,
		command_sub,
		cmd_sub_withbuild,
	},
}
`

type PluginBean struct {
	Name       string
	Version    string
	ExportType string
	Usage      string
}

var pluginTemplate *template.Template

func RenderPluginFile(bean *PluginBean, dest string) (err error) {
	fp, created, err := DirEnsure(dest, true)
	if err != nil {
		return err
	}
	if !created {
		if _, err := os.Stat(fp); err == nil {
			os.Remove(fp)
		}
	}

	var fw *os.File
	fw, err = os.OpenFile(fp, os.O_CREATE|os.O_RDWR, 0644)
	if err == nil {
		defer fw.Close()
		err = pluginTemplate.ExecuteTemplate(fw, "plugin_template", bean)
	}
	return err
}

func init() {
	pt, err := template.New("plugin_template").Parse(tpl)
	if err != nil {
		panic(err)
	}
	pluginTemplate = pt
}

func DirEnsure(dest string, create bool) (abspath string, created bool, err error) {
	tmp := filepath.Dir(dest)
	if !filepath.IsAbs(tmp) {
		tmp, err = filepath.Abs(tmp)
		if err != nil {
			tmp, err = filepath.Abs(fmt.Sprintf("./%s", tmp))
		}
		if err != nil {
			return
		}
	}

	if fdir, e := os.Stat(tmp); e != nil {
		err = e
		if create && os.IsNotExist(err) {
			created = true
			err = os.MkdirAll(tmp, 0777)
		}
	} else if !fdir.IsDir() {
		err = fmt.Errorf("%s不是有效的目录", tmp)
	}

	if err == nil {
		if strings.HasSuffix(dest, "/") {
			abspath = fmt.Sprintf("%s/", tmp)
		} else {
			abspath = fmt.Sprintf("%s/%s", tmp, filepath.Base(dest))
		}
	}
	return
}
