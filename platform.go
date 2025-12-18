package main

import (
	"fmt"
	"log"
	"strings"

	version "github.com/hashicorp/go-version"
)

// Platform is a combination of OS/arch that can be built against.
type Platform struct {
	OS   string
	Arch string

	// Default, if true, will be included as a default build target
	// if no OS/arch is specified. We try to only set as a default popular
	// targets or targets that are generally useful. For example, Android
	// is not a default because it is quite rare that you're cross-compiling
	// something to Android AND something like Linux.
	Default bool
}

func (p *Platform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Arch)
}

// addDrop appends all of the "add" entries and drops the "drop" entries, ignoring
// the "Default" parameter. Uses map for O(n) complexity instead of O(n^2).
func addDrop(base []Platform, add []Platform, drop []Platform) []Platform {
	// Create a map for fast lookup of platforms to drop
	dropMap := make(map[string]bool, len(drop))
	for _, platform := range drop {
		dropMap[platform.String()] = true
	}

	// Create a map to track unique platforms and avoid duplicates
	platformMap := make(map[string]Platform, len(base)+len(add))

	// Add base platforms first, skipping those in drop list
	for _, platform := range base {
		key := platform.String()
		if !dropMap[key] {
			platformMap[key] = platform
		}
	}

	// Add new platforms, skipping those in drop list
	for _, platform := range add {
		key := platform.String()
		if !dropMap[key] {
			platformMap[key] = platform
		}
	}

	// Convert map back to slice
	result := make([]Platform, 0, len(platformMap))
	for _, platform := range platformMap {
		result = append(result, platform)
	}

	return result
}

var (
	Platforms_1_0 = []Platform{
		{"darwin", "386", true},
		{"darwin", "amd64", true},
		{"linux", "386", true},
		{"linux", "amd64", true},
		{"linux", "arm", true},
		{"freebsd", "386", true},
		{"freebsd", "amd64", true},
		{"openbsd", "386", true},
		{"openbsd", "amd64", true},
		{"windows", "386", true},
		{"windows", "amd64", true},
	}

	Platforms_1_1 = addDrop(Platforms_1_0, []Platform{
		{"freebsd", "arm", true},
		{"netbsd", "386", true},
		{"netbsd", "amd64", true},
		{"netbsd", "arm", true},
		{"plan9", "386", false},
	}, nil)

	Platforms_1_3 = addDrop(Platforms_1_1, []Platform{
		{"dragonfly", "386", false},
		{"dragonfly", "amd64", false},
		{"nacl", "amd64", false},
		{"nacl", "amd64p32", false},
		{"nacl", "arm", false},
		{"solaris", "amd64", false},
	}, nil)

	Platforms_1_4 = addDrop(Platforms_1_3, []Platform{
		{"android", "arm", false},
		{"plan9", "amd64", false},
	}, nil)

	Platforms_1_5 = addDrop(Platforms_1_4, []Platform{
		{"darwin", "arm", false},
		{"darwin", "arm64", false},
		{"linux", "arm64", false},
		{"linux", "ppc64", false},
		{"linux", "ppc64le", false},
	}, nil)

	Platforms_1_6 = addDrop(Platforms_1_5, []Platform{
		{"android", "386", false},
		{"android", "amd64", false},
		{"linux", "mips64", false},
		{"linux", "mips64le", false},
		{"nacl", "386", false},
		{"openbsd", "arm", true},
	}, nil)

	Platforms_1_7 = addDrop(Platforms_1_5, []Platform{
		// While not fully supported s390x is generally useful
		{"linux", "s390x", true},
		{"plan9", "arm", false},
		// Add the 1.6 Platforms, but reflect full support for mips64 and mips64le
		{"android", "386", false},
		{"android", "amd64", false},
		{"linux", "mips64", true},
		{"linux", "mips64le", true},
		{"nacl", "386", false},
		{"openbsd", "arm", true},
	}, nil)

	Platforms_1_8 = addDrop(Platforms_1_7, []Platform{
		{"linux", "mips", true},
		{"linux", "mipsle", true},
	}, nil)

	// no new platforms in 1.9
	Platforms_1_9 = Platforms_1_8

	// unannounced, but dropped support for android/amd64
	Platforms_1_10 = addDrop(Platforms_1_9, nil, []Platform{{"android", "amd64", false}})

	Platforms_1_11 = addDrop(Platforms_1_10, []Platform{
		{"js", "wasm", true},
	}, nil)

	Platforms_1_12 = addDrop(Platforms_1_11, []Platform{
		{"aix", "ppc64", false},
		{"windows", "arm", true},
	}, nil)

	Platforms_1_13 = addDrop(Platforms_1_12, []Platform{
		{"illumos", "amd64", false},
		{"netbsd", "arm64", true},
		{"openbsd", "arm64", true},
	}, nil)

	Platforms_1_14 = addDrop(Platforms_1_13, []Platform{
		{"freebsd", "arm64", true},
		{"linux", "riscv64", true},
	}, []Platform{
		// drop nacl
		{"nacl", "386", false},
		{"nacl", "amd64", false},
		{"nacl", "arm", false},
	})

	Platforms_1_15 = addDrop(Platforms_1_14, []Platform{
		{"android", "arm64", false},
	}, []Platform{
		// drop i386 macos
		{"darwin", "386", false},
	})

	Platforms_1_16 = addDrop(Platforms_1_15, []Platform{
		{"android", "amd64", false},
		{"darwin", "arm64", true},
		{"openbsd", "mips64", false},
	}, nil)

	Platforms_1_17 = addDrop(Platforms_1_16, []Platform{
		{"windows", "arm64", true},
	}, nil)

	// no new platforms in 1.18
	Platforms_1_18 = Platforms_1_17

	// Go 1.19: Added linux/loong64 support
	Platforms_1_19 = addDrop(Platforms_1_18, []Platform{
		{"linux", "loong64", true},
	}, nil)

	// Go 1.20: Added freebsd/riscv64 support
	Platforms_1_20 = addDrop(Platforms_1_19, []Platform{
		{"freebsd", "riscv64", true},
	}, nil)

	// Go 1.21: Added android/386, android/arm, and windows/arm64 improvements
	Platforms_1_21 = addDrop(Platforms_1_20, []Platform{
		{"android", "386", false},
		{"android", "arm", false},
		// windows/arm64 was already added in 1.17, but improved in 1.21
	}, nil)

	// Go 1.22: Added darwin/arm64 as default, improved RISC-V support
	Platforms_1_22 = addDrop(Platforms_1_21, []Platform{
		// darwin/arm64 was already added in 1.16, now fully supported as default
	}, nil)

	// Go 1.23: Current latest version, no new architectures but improved existing ones
	Platforms_1_23 = Platforms_1_22

	PlatformsLatest = Platforms_1_23
)

// SupportedPlatforms returns the full list of supported platforms for
// the version of Go that is
func SupportedPlatforms(v string) []Platform {
	// Use latest if we get an unexpected version string
	if !strings.HasPrefix(v, "go") {
		return PlatformsLatest
	}
	// go-version only cares about version numbers
	v = v[2:]

	current, err := version.NewVersion(v)
	if err != nil {
		log.Printf("Unable to parse current go version: %s\n%s", v, err.Error())

		// Default to latest
		return PlatformsLatest
	}

	var platforms = []struct {
		constraint string
		plat       []Platform
	}{
		{"<= 1.0", Platforms_1_0},
		{">= 1.1, < 1.3", Platforms_1_1},
		{">= 1.3, < 1.4", Platforms_1_3},
		{">= 1.4, < 1.5", Platforms_1_4},
		{">= 1.5, < 1.6", Platforms_1_5},
		{">= 1.6, < 1.7", Platforms_1_6},
		{">= 1.7, < 1.8", Platforms_1_7},
		{">= 1.8, < 1.9", Platforms_1_8},
		{">= 1.9, < 1.10", Platforms_1_9},
		{">= 1.10, < 1.11", Platforms_1_10},
		{">= 1.11, < 1.12", Platforms_1_11},
		{">= 1.12, < 1.13", Platforms_1_12},
		{">= 1.13, < 1.14", Platforms_1_13},
		{">= 1.14, < 1.15", Platforms_1_14},
		{">= 1.15, < 1.16", Platforms_1_15},
		{">= 1.16, < 1.17", Platforms_1_16},
		{">= 1.17, < 1.18", Platforms_1_17},
		{">= 1.18, < 1.19", Platforms_1_18},
		{">= 1.19, < 1.20", Platforms_1_19},
		{">= 1.20, < 1.21", Platforms_1_20},
		{">= 1.21, < 1.22", Platforms_1_21},
		{">= 1.22, < 1.23", Platforms_1_22},
		{">= 1.23", Platforms_1_23},
	}

	for _, p := range platforms {
		constraints, err := version.NewConstraint(p.constraint)
		if err != nil {
			panic(err)
		}
		if constraints.Check(current) {
			return p.plat
		}
	}

	// Assume latest
	return PlatformsLatest
}
