// gox 是一个并行交叉编译 Go 应用的工具
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	version "github.com/dudw/gox/internal/version"

	"github.com/dudw/gox/internal/builder"
	"github.com/dudw/gox/internal/platform"
	"github.com/dudw/gox/internal/toolchain"
)

// Version 是 gox 的版本号
const Version = "1.4.0"

func main() {
	os.Exit(realMain())
}

func realMain() int {
	if len(os.Args) >= 2 && os.Args[1] == "version" {
		fmt.Println("gox " + Version)
		return 0
	}

	var buildToolchainFlag bool
	var ldflags string
	var outputTpl string
	var parallel int
	var platformFlag platform.PlatformFlag
	var tags string
	var verbose bool
	var flagGcflags, flagAsmflags string
	var flagCgo, flagRebuild, flagListOSArch, flagRaceFlag, flagTrimpath bool
	var flagGoCmd string
	var modMode string
	flags := flag.NewFlagSet("gox", flag.ExitOnError)
	flags.Usage = func() { printUsage() }
	flags.Var(platformFlag.ArchFlagValue(), "arch", "arch to build for or skip")
	flags.Var(platformFlag.OSArchFlagValue(), "osarch", "os/arch pairs to build for or skip")
	flags.Var(platformFlag.OSFlagValue(), "os", "os to build for or skip")
	flags.StringVar(&ldflags, "ldflags", "", "linker flags")
	flags.StringVar(&tags, "tags", "", "go build tags")
	flags.StringVar(&outputTpl, "output", "{{.Dir}}_{{.OS}}_{{.Arch}}", "output path")
	flags.IntVar(&parallel, "parallel", -1, "parallelization factor")
	flags.BoolVar(&buildToolchainFlag, "build-toolchain", false, "build toolchain")
	flags.BoolVar(&verbose, "verbose", false, "verbose")
	flags.BoolVar(&flagCgo, "cgo", false, "")
	flags.BoolVar(&flagRebuild, "rebuild", false, "")
	flags.BoolVar(&flagListOSArch, "osarch-list", false, "")
	flags.BoolVar(&flagRaceFlag, "race", false, "")
	flags.BoolVar(&flagTrimpath, "trimpath", false, "")
	flags.StringVar(&flagGcflags, "gcflags", "", "")
	flags.StringVar(&flagAsmflags, "asmflags", "", "")
	flags.StringVar(&flagGoCmd, "gocmd", "go", "")
	flags.StringVar(&modMode, "mod", "", "")
	if err := flags.Parse(os.Args[1:]); err != nil {
		flags.Usage()
		return 1
	}

	if parallel <= 0 {
		cpus := runtime.NumCPU()
		if cpus < 2 {
			parallel = 1
		} else {
			parallel = cpus - 1
		}

		if runtime.GOOS == "solaris" {
			parallel = 3
		}
	}

	if buildToolchainFlag {
		return toolchain.MainBuildToolchain(parallel, platformFlag, verbose)
	}

	if _, err := exec.LookPath(flagGoCmd); err != nil {
		fmt.Fprintf(os.Stderr, "%s executable must be on the PATH\n",
			flagGoCmd)
		return 1
	}

	versionStr, err := builder.GoVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading Go version: %s", err)
		return 1
	}

	if flagListOSArch {
		return mainListOSArch(versionStr)
	}

	packages := flags.Args()
	if len(packages) == 0 {
		packages = []string{"."}
	}

	mainDirs, err := builder.GoMainDirs(packages, flagGoCmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading packages: %s", err)
		return 1
	}

	platforms := platformFlag.Platforms(platform.SupportedPlatforms(versionStr))
	if len(platforms) == 0 {
		fmt.Println("No valid platforms to build for. If you specified a value")
		fmt.Println("for the 'os', 'arch', or 'osarch' flags, make sure you're")
		fmt.Println("using a valid value.")
		return 1
	}

	if modMode != "" && strings.HasPrefix(versionStr, "go") {
		current, err := version.NewVersion(versionStr[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse current go version: %s\n%s", versionStr, err.Error())
			return 1
		}

		constraint, err := version.NewConstraint(">= 1.11")
		if err != nil {
			panic(err)
		}

		if !constraint.Check(current) {
			fmt.Printf("Go compiler version %s does not support the -mod flag\n", versionStr)
			modMode = ""
		}
	}

	fmt.Printf("Number of parallel builds: %d\n\n", parallel)
	var errorLock sync.Mutex
	var wg sync.WaitGroup
	errors := make([]string, 0)
	semaphore := make(chan int, parallel)
	for _, p := range platforms {
		for _, path := range mainDirs {
			wg.Add(1)
			go func(path string, p platform.Platform) {
				defer wg.Done()
				semaphore <- 1
				fmt.Printf("--> %15s: %s\n", p.String(), path)

				opts := &builder.CompileOpts{
					PackagePath: path,
					Platform:    p,
					OutputTpl:   outputTpl,
					Ldflags:     ldflags,
					Gcflags:     flagGcflags,
					Asmflags:    flagAsmflags,
					Tags:        tags,
					ModMode:     modMode,
					Cgo:         flagCgo,
					Rebuild:     flagRebuild,
					GoCmd:       flagGoCmd,
					Race:        flagRaceFlag,
					Trimpath:    flagTrimpath,
				}

				builder.EnvOverride(&opts.Ldflags, p, "LDFLAGS")
				builder.EnvOverride(&opts.Gcflags, p, "GCFLAGS")
				builder.EnvOverride(&opts.Asmflags, p, "ASMFLAGS")

				if err := builder.GoCrossCompile(opts); err != nil {
					errorLock.Lock()
					defer errorLock.Unlock()
					errors = append(errors,
						fmt.Sprintf("%s error: %s", p.String(), err))
				}
				<-semaphore
			}(path, p)
		}
	}
	wg.Wait()

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d errors occurred:\n", len(errors))
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "--> %s\n", err)
		}
		return 1
	}

	return 0
}

func mainListOSArch(versionStr string) int {
	fmt.Printf(
		"Supported OS/Arch combinations for %s are shown below. The \"default\"\n"+
			"boolean means that if you don't specify an OS/Arch, it will be\n"+
			"included by default. If it isn't a default OS/Arch, you must explicitly\n"+
			"specify that OS/Arch combo for Gox to use it.\n\n",
		versionStr)
	for _, p := range platform.SupportedPlatforms(versionStr) {
		fmt.Printf("%s\t(default: %v)\n", p.String(), p.Default)
	}

	return 0
}

func printUsage() {
	fmt.Fprintf(os.Stderr, helpText)
}

const helpText = `Usage: gox [options] [packages]

  Gox cross-compiles Go applications in parallel.

  If no specific operating systems or architectures are specified, Gox
  will build for all pairs supported by your version of Go.
  
  If 'version' is specified as the first argument, Gox will print
  the version of gox.

Options:

  -arch=""            Space-separated list of architectures to build for
  -build-toolchain    Build cross-compilation toolchain
  -cgo                Sets CGO_ENABLED=1, requires proper C toolchain (advanced)
  -gcflags=""         Additional '-gcflags' value to pass to go build
  -ldflags=""         Additional '-ldflags' value to pass to go build
  -asmflags=""        Additional '-asmflags' value to pass to go build
  -tags=""            Additional '-tags' value to pass to go build
  -mod=""             Additional '-mod' value to pass to go build
  -os=""              Space-separated list of operating systems to build for
  -osarch=""          Space-separated list of os/arch pairs to build for
  -osarch-list        List supported os/arch pairs for your Go version
  -output="foo"       Output path template. See below for more info
  -parallel=-1        Amount of parallelism, defaults to number of CPUs
  -race               Build with the go race detector enabled, requires CGO
  -trimpath           Remove all file system paths from the resulting executable
  -gocmd="go"         Build command, defaults to Go
  -rebuild            Force rebuilding of package that were up to date
  -verbose            Verbose mode

Output path template:

  The output path for the compiled binaries is specified with the
  "-output" flag. The value is a string that is a Go text template.
  The default value is "{{.Dir}}_{{.OS}}_{{.Arch}}". The variables and
  their values should be self-explanatory.

Platforms (OS/Arch):

  The operating systems and architectures to cross-compile for may be
  specified with the "-arch" and "-os" flags. These are space separated lists
  of valid GOOS/GOARCH values to build for, respectively. You may prefix an
  OS or Arch with "!" to negate and not build for that platform. If the list
  is made up of only negations, then the negations will come from the default
  list.

  Additionally, the "-osarch" flag may be used to specify complete os/arch
  pairs that should be built or ignored. The syntax for this is what you would
  expect: "darwin/amd64" would be a valid osarch value. Multiple can be space
  separated. An os/arch pair can begin with "!" to not build for that platform.

  The "-osarch" flag has the highest precedent when determing whether to
  build for a platform. If it is included in the "-osarch" list, it will be
  built even if the specific os and arch is negated in "-os" and "-arch",
  respectively.

Platform Overrides:

  The "-gcflags", "-ldflags" and "-asmflags" options can be overridden per-platform
  by using environment variables. Gox will look for environment variables
  in the following format and use those to override values if they exist:

    GOX_[OS]_[ARCH]_GCFLAGS
    GOX_[OS]_[ARCH]_LDFLAGS
    GOX_[OS]_[ARCH]_ASMFLAGS

`
