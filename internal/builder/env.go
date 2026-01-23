package builder

import (
	"fmt"
	"os"
	"strings"

	"github.com/dudw/gox/internal/platform"
)

// EnvOverride 根据 GOX_{OS}_{ARCH}_{KEY} 格式的环境变量覆盖目标值
func EnvOverride(target *string, p platform.Platform, key string) {
	key = strings.ToUpper(fmt.Sprintf(
		"GOX_%s_%s_%s", p.OS, p.Arch, key))
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}
