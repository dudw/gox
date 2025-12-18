package main

import (
	"flag"
	"fmt"
	"strings"
)

// PlatformFlag is a flag.Value (and flag.Getter) implementation that
// is used to track the os/arch flags on the command-line.
type PlatformFlag struct {
	OS     []string
	Arch   []string
	OSArch []Platform
}

// Platforms returns the list of platforms that were set by this flag.
// The default set of platforms must be passed in.
func (p *PlatformFlag) Platforms(supported []Platform) []Platform {
	// Build inclusion and exclusion maps for efficient lookup
	ignoreArch := make(map[string]bool)
	includeArch := make(map[string]bool)
	ignoreOS := make(map[string]bool)
	includeOS := make(map[string]bool)
	ignoreOSArch := make(map[string]bool)
	includeOSArch := make(map[string]bool)

	// Parse arch flags
	for _, v := range p.Arch {
		if v[0] == '!' {
			ignoreArch[v[1:]] = true
		} else {
			includeArch[v] = true
		}
	}

	// Parse OS flags
	for _, v := range p.OS {
		if v[0] == '!' {
			ignoreOS[v[1:]] = true
		} else {
			includeOS[v] = true
		}
	}

	// Parse OS/Arch pairs
	for _, v := range p.OSArch {
		if v.OS[0] == '!' {
			platform := Platform{OS: v.OS[1:], Arch: v.Arch}
			ignoreOSArch[platform.String()] = true
		} else {
			includeOSArch[v.String()] = true
		}
	}

	// Create a map of supported platforms for fast lookup
	supportedMap := make(map[string]Platform, len(supported))
	for _, platform := range supported {
		supportedMap[platform.String()] = platform
	}

	// Determine which platforms to build
	result := make([]Platform, 0)

	// If specific OS/Arch pairs are specified, use those
	if len(includeOSArch) > 0 {
		for platformStr := range includeOSArch {
			if platform, exists := supportedMap[platformStr]; exists && !ignoreOSArch[platformStr] {
				platform.Default = false
				result = append(result, platform)
			}
		}
	} else if len(includeOS) > 0 && len(includeArch) > 0 {
		// Build combinations of specified OS and Arch
		for os := range includeOS {
			for arch := range includeArch {
				platform := Platform{OS: os, Arch: arch}
				if _, exists := supportedMap[platform.String()]; exists {
					platform.Default = false
					result = append(result, platform)
				}
			}
		}
	} else if len(includeOS) > 0 {
		// Use specified OS with all supported architectures
		for os := range includeOS {
			for _, platform := range supported {
				if platform.OS == os {
					platform.Default = false
					result = append(result, platform)
				}
			}
		}
	} else {
		// Use default platforms
		for _, platform := range supported {
			if platform.Default {
				platform.Default = false
				result = append(result, platform)
			}
		}
	}

	// Apply exclusion filters
	filteredResult := make([]Platform, 0, len(result))
	for _, platform := range result {
		platformStr := platform.String()

		// Skip if explicitly excluded via OS/Arch pair
		if ignoreOSArch[platformStr] {
			continue
		}

		// Skip if excluded via individual OS or Arch
		if ignoreOS[platform.OS] || ignoreArch[platform.Arch] {
			continue
		}

		// Skip if not included via individual OS or Arch (when no OS/Arch pairs specified)
		if len(includeOSArch) == 0 && len(includeOS) > 0 && !includeOS[platform.OS] {
			continue
		}
		if len(includeOSArch) == 0 && len(includeArch) > 0 && !includeArch[platform.Arch] {
			continue
		}

		filteredResult = append(filteredResult, platform)
	}

	return filteredResult
}

// ArchFlagValue returns a flag.Value that can be used with the flag
// package to collect the arches for the flag.
func (p *PlatformFlag) ArchFlagValue() flag.Value {
	return (*appendStringValue)(&p.Arch)
}

// OSFlagValue returns a flag.Value that can be used with the flag
// package to collect the operating systems for the flag.
func (p *PlatformFlag) OSFlagValue() flag.Value {
	return (*appendStringValue)(&p.OS)
}

// OSArchFlagValue returns a flag.Value that can be used with the flag
// package to collect complete os and arch pairs for the flag.
func (p *PlatformFlag) OSArchFlagValue() flag.Value {
	return (*appendPlatformValue)(&p.OSArch)
}

// appendPlatformValue is a flag.Value that appends a full platform (os/arch)
// to a list where the values from space-separated lines. This is used to
// satisfy the -osarch flag.
type appendPlatformValue []Platform

func (s *appendPlatformValue) String() string {
	return ""
}

func (s *appendPlatformValue) Set(value string) error {
	if value == "" {
		return nil
	}

	for _, v := range strings.Split(value, " ") {
		parts := strings.Split(v, "/")
		if len(parts) != 2 {
			return fmt.Errorf(
				"Invalid platform syntax: %s should be os/arch", v)
		}

		platform := Platform{
			OS:   strings.ToLower(parts[0]),
			Arch: strings.ToLower(parts[1]),
		}

		s.appendIfMissing(&platform)
	}

	return nil
}

func (s *appendPlatformValue) appendIfMissing(value *Platform) {
	for _, existing := range *s {
		if existing == *value {
			return
		}
	}

	*s = append(*s, *value)
}

// appendStringValue is a flag.Value that appends values to the list,
// where the values come from space-separated lines. This is used to
// satisfy the -os="windows linux" flag to become []string{"windows", "linux"}
type appendStringValue []string

func (s *appendStringValue) String() string {
	return strings.Join(*s, " ")
}

func (s *appendStringValue) Set(value string) error {
	for _, v := range strings.Split(value, " ") {
		if v != "" {
			s.appendIfMissing(strings.ToLower(v))
		}
	}

	return nil
}

func (s *appendStringValue) appendIfMissing(value string) {
	for _, existing := range *s {
		if existing == value {
			return
		}
	}

	*s = append(*s, value)
}
