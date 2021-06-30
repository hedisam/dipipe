package pluginbuilder

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"
)

func TestBuild(t *testing.T) {
	// Docker is needed to run this test

	// same as 'pwd' linux command
	workDir, err := os.Getwd()
	if err != nil {
		t.Errorf("failed to get the current directory path: %v", err)
		return
	}

	name := "myplugin"
	// path to the plugin's code directory. it needs to be an absolute path.
	srcPath := path.Join(workDir, "pluginsrctest")
	pluginPath, err := Build(context.Background(), name, srcPath)
	if err != nil {
		t.Errorf("failed to generate the plugin: %v", err)
		return
	}

	_, err = os.Stat(pluginPath)
	if err != nil {
		t.Errorf("invalid path returned/not exists, got: %s, err: %v", pluginPath, err)
		return
	}

	if !strings.HasSuffix(pluginPath, name+".so") {
		t.Errorf("expected to get a path ending with %s, got: %s", name+".so", pluginPath)
	}
}