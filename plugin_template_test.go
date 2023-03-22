package gocli

import "testing"

func TestRenderFile(t *testing.T) {
	err := RenderPluginFile(&PluginBean{
		Name:    "demo",
		Version: "v0.0.1",
	}, "../plugin/demo/plugin.go")
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	t.Log("succ!")
}
