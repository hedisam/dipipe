package pluginbuilder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

// Build attempts to build a plugin (linkable .so file) from a source code directory.
// srcPath is the absolute path to the user's code directory.
// name is used as the name of the compiled plugin file (e.g. name: "hello" -> "hello.so") as well as the name of
// the docker image which is responsible for compiling the plugin code.
// It returns the path of the built plugin file.
func Build(ctx context.Context, name, srcPath string) (string, error) {
	// 2. copy the plugin builder's Dockerfile to dipipe-build
	err := writeDockerfile(srcPath)
	if err != nil {
		return "", fmt.Errorf("pluginbuilder: Build: failed to write docker file in the user's code dir: %w", err)
	}

	// 3. run go mod init & go mod vendor in dipipe-build
	err = initModVendor(ctx, srcPath)
	if err != nil {
		return "", fmt.Errorf("pluginbuilder: Build: failed to enable go modules for the src directory: %w", err)
	}

	// 4. compile & build the plugin using Docker image builder
	imageName, err := buildPluginImage(ctx, srcPath, name)
	if err != nil {
		return "", fmt.Errorf("pluginbuilder: Build: failed to build the docker image: %w", err)
	}

	// 5. run the image to extract the plugin
	pluginPath, err := extractPlugin(ctx, srcPath, name, imageName)
	if err != nil {
		return "", fmt.Errorf("pluginbuilder: Build: failed to extract the plugin: %w", err)
	}
	return pluginPath, nil
}

func extractPlugin(ctx context.Context, dir, pluginName, imageName string) (string, error) {
	// we could use the docker SDK for golang but this is straightforward enough.
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-v", dir+":/app/plugin/", imageName)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run the plugin extractor container: %w", err)
	}

	// check if the plugin has been extracted
	pluginPath := path.Join(dir, pluginName+".so")
	ok, msg, err := pathExists(pluginPath)
	if err != nil {
		return "", fmt.Errorf("plugin existence check failed: %w", err)
	} else if !ok {
		return "", fmt.Errorf("plugin not extracted: %s", msg)
	}

	return pluginPath, nil
}

func buildPluginImage(ctx context.Context, dir, name string) (string, error) {
	// we could use the docker SDK for golang but this is straightforward enough.
	errW := new(stdErr)
	imageName := "dipipe-"+name+":1.0.0"
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageName, "--build-arg", "plugin_name="+name, ".")
	cmd.Dir = dir
	cmd.Stderr = errW
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to build the pluginbuilder's docker image: %w, stderr: %s", err, errW.err)
	}

	return imageName, nil
}

func initModVendor(ctx context.Context, dir string) error {
	// run 'go mod init'
	errW := new(stdErr)
	cmd := exec.Command("go", "mod", "init")
	cmd.Dir = dir
	cmd.Stderr = errW
	err := cmd.Run()
	if err != nil && !strings.Contains(errW.err, "already exists") {
		return fmt.Errorf("failed to run the command 'go mod init': %w", err)
	}

	// run 'go mod tidy'
	cmd = exec.CommandContext(ctx,"go", "mod", "tidy")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run the command 'go mod tidy': %w", err)
	}

	// run 'go mod vendor'
	cmd = exec.CommandContext(ctx, "go", "mod", "vendor")
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run the command 'go mod vendor': %w", err)
	}

	return nil
}

func writeDockerfile(dstPath string) error {
	dst, err := os.Create(path.Join(dstPath, "Dockerfile"))
	if err != nil {
		return fmt.Errorf("failed to create an empty Dockerfile in the dst directory: %w", err)
	}
	defer dst.Close()

	_, err = dst.WriteString(dockerfile)
	if err != nil {
		return fmt.Errorf("failed to write the Dockerfile content: %w", err)
	}

	return nil
}

func pathExists(path string) (bool, string, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, err.Error(), nil
		}
		return false, "", fmt.Errorf("failed to check %s existence: %w", path, err)
	}

	return true, "", nil
}

type stdErr struct {
	err string
}

func (s *stdErr) Write(p []byte) (n int, err error) {
	s.err += string(p) + "\n"
	return len(p), nil
}

var dockerfile = `FROM golang:1.16.5-alpine
ARG plugin_name
ENV env_plugin_name "$plugin_name.so"
RUN apk add build-base
RUN mkdir -p /app/plugin
WORKDIR /app
COPY . .
RUN GOOS=linux go build -buildmode=plugin -o ${plugin_name}.so .
CMD ["sh", "-c", "cp $env_plugin_name plugin/$env_plugin_name"]
`