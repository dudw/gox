// Package builder 提供 Go 交叉编译功能
package builder

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/dudw/gox/internal/platform"
)

// OutputTemplateData 用于输出路径模板的数据
type OutputTemplateData struct {
	Dir  string
	OS   string
	Arch string
}

// CompileOpts 编译选项
type CompileOpts struct {
	PackagePath string
	Platform    platform.Platform
	OutputTpl   string
	Ldflags     string
	Gcflags     string
	Asmflags    string
	Tags        string
	ModMode     string
	Cgo         bool
	Rebuild     bool
	GoCmd       string
	Race        bool
	Trimpath    bool
}

// GoCrossCompile 执行交叉编译
func GoCrossCompile(opts *CompileOpts) error {
	env := append(os.Environ(),
		"GOOS="+opts.Platform.OS,
		"GOARCH="+opts.Platform.Arch)

	if !opts.Cgo && os.Getenv("CGO_ENABLED") != "0" {
		opts.Cgo = runtime.GOOS == opts.Platform.OS &&
			runtime.GOARCH == opts.Platform.Arch
	}

	if opts.Cgo {
		env = append(env, "CGO_ENABLED=1")
	} else {
		env = append(env, "CGO_ENABLED=0")
	}

	var outputPath bytes.Buffer
	tpl, err := template.New("output").Parse(opts.OutputTpl)
	if err != nil {
		return err
	}
	tplData := OutputTemplateData{
		Dir:  filepath.Base(opts.PackagePath),
		OS:   opts.Platform.OS,
		Arch: opts.Platform.Arch,
	}
	if err := tpl.Execute(&outputPath, &tplData); err != nil {
		return err
	}

	if opts.Platform.OS == "windows" {
		outputPath.WriteString(".exe")
	}

	outputPathReal := outputPath.String()
	outputPathReal, err = filepath.Abs(outputPathReal)
	if err != nil {
		return err
	}

	chdir := ""
	if opts.PackagePath[0] == '_' {
		if runtime.GOOS == "windows" {
			re := regexp.MustCompile("^/([a-zA-Z])_/")
			chdir = re.ReplaceAllString(opts.PackagePath[1:], "$1:\\")
			chdir = strings.Replace(chdir, "/", "\\", -1)
		} else {
			chdir = opts.PackagePath[1:]
		}

		opts.PackagePath = ""
	}

	args := []string{"build"}
	if opts.Rebuild {
		args = append(args, "-a")
	}
	if opts.ModMode != "" {
		args = append(args, "-mod", opts.ModMode)
	}
	if opts.Race {
		args = append(args, "-race")
	}
	if opts.Trimpath {
		args = append(args, "-trimpath")
	}
	args = append(args,
		"-gcflags", opts.Gcflags,
		"-ldflags", opts.Ldflags,
		"-asmflags", opts.Asmflags,
		"-tags", opts.Tags,
		"-o", outputPathReal,
		opts.PackagePath)

	_, err = execGo(opts.GoCmd, env, chdir, args...)
	return err
}

// GoMainDirs 返回 main 包的路径列表
func GoMainDirs(packages []string, GoCmd string) ([]string, error) {
	args := make([]string, 0, len(packages)+3)
	args = append(args, "list", "-f", "{{.Name}}|{{.ImportPath}}")
	args = append(args, packages...)

	output, err := execGo(GoCmd, nil, "", args...)
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(output))
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			log.Printf("Bad line reading packages: %s", line)
			continue
		}

		if parts[0] == "main" {
			results = append(results, parts[1])
		}
	}

	return results, nil
}

// GoRoot 返回 GOROOT 路径
func GoRoot() (string, error) {
	output, err := execGo("go", nil, "", "env", "GOROOT")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

// GoVersion 读取 PATH 上 go 二进制的版本
func GoVersion() (string, error) {
	td, err := os.MkdirTemp("", "gox")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(td)

	sourcePath := filepath.Join(td, "version.go")
	if err := os.WriteFile(sourcePath, []byte(versionSource), 0644); err != nil {
		return "", err
	}

	return execGo("go", nil, "", "run", sourcePath)
}

// GoVersionParts 解析版本号为 major 和 minor
func GoVersionParts() (result [2]int, err error) {
	version, err := GoVersion()
	if err != nil {
		return
	}

	_, err = fmt.Sscanf(version, "go%d.%d", &result[0], &result[1])
	return
}

func execGo(GoCmd string, env []string, dir string, args ...string) (string, error) {
	var stderr, stdout bytes.Buffer
	cmd := exec.Command(GoCmd, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if env != nil {
		cmd.Env = env
	}
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("%s\nStderr: %s", err, stderr.String())
		return "", err
	}

	return stdout.String(), nil
}

const versionSource = `package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Print(runtime.Version())
}`
