package platform

import (
	"flag"
	"fmt"
	"strings"
)

// PlatformFlag 用于跟踪命令行的 os/arch 标志
type PlatformFlag struct {
	OS     []string
	Arch   []string
	OSArch []Platform
}

// Platforms 返回由此标志设置的平台列表
func (p *PlatformFlag) Platforms(supported []Platform) []Platform {
	ignoreArch := make(map[string]bool)
	includeArch := make(map[string]bool)
	ignoreOS := make(map[string]bool)
	includeOS := make(map[string]bool)
	ignoreOSArch := make(map[string]bool)
	includeOSArch := make(map[string]bool)

	for _, v := range p.Arch {
		if v[0] == '!' {
			ignoreArch[v[1:]] = true
		} else {
			includeArch[v] = true
		}
	}

	for _, v := range p.OS {
		if v[0] == '!' {
			ignoreOS[v[1:]] = true
		} else {
			includeOS[v] = true
		}
	}

	for _, v := range p.OSArch {
		if v.OS[0] == '!' {
			platform := Platform{OS: v.OS[1:], Arch: v.Arch}
			ignoreOSArch[platform.String()] = true
		} else {
			includeOSArch[v.String()] = true
		}
	}

	supportedMap := make(map[string]Platform, len(supported))
	for _, platform := range supported {
		supportedMap[platform.String()] = platform
	}

	result := make([]Platform, 0)
	added := make(map[string]bool)

	// 首先添加 OSArch 中明确包含的平台（按 p.OSArch 顺序）
	for _, v := range p.OSArch {
		if v.OS[0] == '!' {
			continue
		}
		platformStr := v.String()
		if platform, exists := supportedMap[platformStr]; exists {
			if !added[platformStr] {
				platform.Default = false
				result = append(result, platform)
				added[platformStr] = true
			}
		}
	}

	// 然后处理 OS 和 Arch 的组合
	if len(includeOS) > 0 && len(includeArch) > 0 {
		for _, platform := range supported {
			if includeOS[platform.OS] && includeArch[platform.Arch] {
				platformStr := platform.String()
				if !added[platformStr] {
					platform.Default = false
					result = append(result, platform)
					added[platformStr] = true
				}
			}
		}
	} else if len(includeOS) > 0 && len(includeOSArch) == 0 {
		for _, platform := range supported {
			if includeOS[platform.OS] {
				platformStr := platform.String()
				if !added[platformStr] {
					platform.Default = false
					result = append(result, platform)
					added[platformStr] = true
				}
			}
		}
	} else if len(includeOSArch) == 0 {
		// 没有指定任何内容时使用默认平台
		for _, platform := range supported {
			if platform.Default {
				platformStr := platform.String()
				if !added[platformStr] {
					platform.Default = false
					result = append(result, platform)
					added[platformStr] = true
				}
			}
		}
	}

	// 应用排除规则
	filteredResult := make([]Platform, 0, len(result))
	for _, platform := range result {
		platformStr := platform.String()

		// 排除 OSArch 中明确忽略的
		if ignoreOSArch[platformStr] {
			continue
		}

		// 排除 OS 或 Arch 被忽略的
		if ignoreOS[platform.OS] || ignoreArch[platform.Arch] {
			continue
		}

		filteredResult = append(filteredResult, platform)
	}

	return filteredResult
}

// ArchFlagValue 返回用于收集 arch 标志的 flag.Value
func (p *PlatformFlag) ArchFlagValue() flag.Value {
	return (*appendStringValue)(&p.Arch)
}

// OSFlagValue 返回用于收集 os 标志的 flag.Value
func (p *PlatformFlag) OSFlagValue() flag.Value {
	return (*appendStringValue)(&p.OS)
}

// OSArchFlagValue 返回用于收集 os/arch 对的 flag.Value
func (p *PlatformFlag) OSArchFlagValue() flag.Value {
	return (*appendPlatformValue)(&p.OSArch)
}

// appendPlatformValue 是用于追加完整平台 (os/arch) 的 flag.Value
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

// appendStringValue 用于追加空格分隔的值
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
