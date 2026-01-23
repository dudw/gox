// Package toolchain 提供 Go 工具链构建功能
package toolchain

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/dudw/gox/internal/builder"
	"github.com/dudw/gox/internal/iochan"
	"github.com/dudw/gox/internal/platform"
)

// MainBuildToolchain 构建工具链的主函数
func MainBuildToolchain(parallel int, platformFlag platform.PlatformFlag, verbose bool) int {
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintf(os.Stderr, "You must have Go already built for your native platform\n")
		fmt.Fprintf(os.Stderr, "and the `go` binary on the PATH to build toolchains.\n")
		return 1
	}

	versionParts, err := builder.GoVersionParts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading Go version: %s", err)
		return 1
	}
	if versionParts[0] >= 1 && versionParts[1] >= 5 {
		fmt.Fprintf(
			os.Stderr,
			"-build-toolchain is no longer required for Go 1.5 or later.\n"+
				"You can start using Gox immediately!\n")
		return 1
	}

	version, err := builder.GoVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading Go version: %s", err)
		return 1
	}

	root, err := builder.GoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error finding GOROOT: %s\n", err)
		return 1
	}

	if verbose {
		fmt.Println("Verbose mode enabled. Output from building each toolchain will be")
		fmt.Println("outputted to stdout as they are built.\n ")
	}

	platforms := platformFlag.Platforms(platform.SupportedPlatforms(version))

	if parallel > 1 {
		fmt.Println("The toolchain build can't be parallelized because compiling a single")
		fmt.Println("Go source directory can only be done for one platform at a time. Therefore,")
		fmt.Println("the toolchain for each platform will be built one at a time.\n ")
	}
	parallel = 1

	var errorLock sync.Mutex
	var wg sync.WaitGroup
	errs := make([]error, 0)
	semaphore := make(chan int, parallel)
	for _, p := range platforms {
		wg.Add(1)
		go func(p platform.Platform) {
			err := buildToolchain(&wg, semaphore, root, p, verbose)
			if err != nil {
				errorLock.Lock()
				defer errorLock.Unlock()
				errs = append(errs, fmt.Errorf("%s: %s", p.String(), err))
			}
		}(p)
	}
	wg.Wait()

	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d errors occurred:\n", len(errs))
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		return 1
	}

	return 0
}

func buildToolchain(wg *sync.WaitGroup, semaphore chan int, root string, p platform.Platform, verbose bool) error {
	defer wg.Done()
	semaphore <- 1
	defer func() { <-semaphore }()
	fmt.Printf("--> Toolchain: %s\n", p.String())

	scriptName := "make.bash"
	if runtime.GOOS == "windows" {
		scriptName = "make.bat"
	}

	var stderr bytes.Buffer
	var stdout bytes.Buffer
	scriptDir := filepath.Join(root, "src")
	scriptPath := filepath.Join(scriptDir, scriptName)
	cmd := exec.Command(scriptPath, "--no-clean")
	cmd.Dir = scriptDir
	cmd.Env = append(os.Environ(),
		"GOARCH="+p.Arch,
		"GOOS="+p.OS)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if verbose {
		r, w := io.Pipe()
		cmd.Stdout = w
		cmd.Stderr = io.MultiWriter(cmd.Stderr, w)

		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			for line := range iochan.DelimReader(r, '\n') {
				fmt.Printf("%s: %s", p.String(), line)
			}
		}()
		defer func() {
			w.Close()
			<-doneCh
		}()
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Error building '%s': %s", p.String(), err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Error building '%s'.\n\nStdout: %s\n\nStderr: %s\n",
			p.String(), stdout.String(), stderr.String())
	}

	return nil
}
