# Gox - Go 交叉编译工具

## 项目概述

Gox 是一个简单、无额外依赖的 Go 交叉编译工具，其行为与标准的 `go build` 非常相似。Gox 能够并行构建多个平台，并会自动构建交叉编译工具链（针对 Go 1.5 之前的版本）。

**核心特性：**
- 并行构建：根据 CPU 核心数自动并行化构建过程
- 平台过滤：支持通过 `-os`、`-arch`、`-osarch` 标志筛选构建目标
- 输出模板：支持自定义输出路径模板
- 环境变量覆盖：支持通过环境变量覆盖特定平台的编译标志
- 向后兼容：支持从 Go 1.0 到最新版本的所有平台

**技术栈：**
- 语言：Go 1.25+
- 依赖：
  - `github.com/hashicorp/go-version` (版本约束处理)
  - `github.com/mitchellh/iochan` (I/O 通道工具)

## 安装

```bash
# 安装最新版本
go install github.com/mitchellh/gox@latest

# 验证安装
gox -h
```

## 使用方法

### 基本使用

```bash
# 构建当前包的所有平台
gox

# 构建当前包及其子包
gox ./...

# 构建特定包
gox github.com/mitchellh/gox github.com/hashicorp/serf

# 仅构建 Linux 平台
gox -os="linux"

# 仅构建 64 位 Linux
gox -osarch="linux/amd64"
```

### 常用选项

| 选项 | 描述 | 默认值 |
|------|------|--------|
| `-arch` | 架构列表（空格分隔） | 所有支持的架构 |
| `-os` | 操作系统列表（空格分隔） | 所有支持的系统 |
| `-osarch` | OS/架构对列表（空格分隔） | 所有支持的平台 |
| `-output` | 输出路径模板 | `{{.Dir}}_{{.OS}}_{{.Arch}}` |
| `-parallel` | 并行构建数 | CPU 数-1 |
| `-ldflags` | 链接器标志 | 空 |
| `-tags` | 构建标签 | 空 |
| `-build-toolchain` | 构建工具链（Go <1.5） | false |
| `-verbose` | 详细模式 | false |
| `-osarch-list` | 列出支持的平台 | false |

### 输出模板

输出路径使用 Go 文本模板，可用变量：
- `{{.Dir}}`: 包目录名
- `{{.OS}}`: 目标操作系统
- `{{.Arch}}`: 目标架构

示例：
```bash
# 输出到 dist/目录
gox -output="dist/{{.Dir}}_{{.OS}}_{{.Arch}}"

# 输出到带时间戳的目录
gox -output="build/{{.Dir}}/{{.OS}}/{{.Arch}}/{{.Dir}}"
```

### 平台过滤

支持否定过滤：
```bash
# 构建除 Windows 外的所有平台
gox -os="!windows"

# 构建所有 64 位架构，但排除 arm64
gox -arch="amd64 386 !arm64"

# 使用 osarch 精确控制
gox -osarch="linux/amd64 darwin/amd64 !windows/386"
```

### 环境变量覆盖

可以通过环境变量覆盖特定平台的编译标志：
- `GOX_[OS]_[ARCH]_GCFLAGS`: 覆盖 `-gcflags`
- `GOX_[OS]_[ARCH]_LDFLAGS`: 覆盖 `-ldflags`
- `GOX_[OS]_[ARCH]_ASMFLAGS`: 覆盖 `-asmflags`

示例：
```bash
# 为 Linux/ARM 设置特定的 LDFLAGS
export GOX_LINUX_ARM_LDFLAGS="-X main.version=1.0.0"
gox -osarch="linux/arm"
```

## 构建和测试

### 构建项目

```bash
# 使用标准 Go 构建
go build -o gox .

# 使用 Gox 自举构建（跨平台）
gox -output="dist/gox_{{.OS}}_{{.Arch}}"
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test -v ./platform_test.go
```

### 预构建脚本

Windows 环境下的构建脚本 (`build.cmd`)：
```batch
gox -ldflags="-s -w" -osarch="linux/amd64 windows/amd64" -output="dist/{{.Dir}}_{{.OS}}_{{.Arch}}"
```

## 开发约定

### 代码结构

```
.
├── main.go              # 主程序入口，命令行参数解析
├── go.go               # 核心 Go 工具链交互逻辑
├── platform.go         # 平台定义和支持版本管理
├── platform_flag.go    # 平台标志解析器
├── toolchain.go        # 工具链构建逻辑（Go <1.5）
├── env_override.go     # 环境变量覆盖逻辑
├── main_osarch.go      # 主平台列表输出
├── go_test.go          # 核心功能测试
├── platform_test.go    # 平台逻辑测试
├── platform_flag_test.go # 平台标志测试
└── build.cmd           # Windows 构建脚本
```

### 平台支持管理

平台定义在 `platform.go` 中，按 Go 版本组织。每个 Go 版本都有对应的平台列表，包含默认构建目标。

添加新平台时：
1. 在适当的 `Platforms_X_X` 切片中添加新 `Platform` 结构
2. 更新 `SupportedPlatforms` 函数中的版本约束
3. 更新 `PlatformsLatest` 变量

### 错误处理

- 使用 `os.Exit(realMain())` 确保 `defer` 正确执行
- 并行构建错误通过通道收集并统一显示
- 工具链构建错误包含详细的标准输出和错误输出

### 测试覆盖

- 单元测试覆盖核心功能
- 平台逻辑测试验证版本匹配
- 标志解析测试确保正确过滤

## 项目结构详解

### 核心文件

1. **main.go**: 命令行界面，参数解析，并行构建调度
2. **go.go**: 
   - `GoCrossCompile`: 执行交叉编译
   - `GoMainDirs`: 查找主包目录
   - `GoVersion`: 读取 Go 版本
   - `execGo`: 执行 Go 命令的辅助函数
3. **platform.go**: 
   - `Platform` 结构定义
   - `SupportedPlatforms`: 根据 Go 版本返回支持的平台
   - 各版本平台列表（1.0 到最新）
4. **platform_flag.go**: 
   - `PlatformFlag` 类型和值解析
   - 支持否定过滤和优先级规则

### 依赖管理

- **go-version**: 用于解析和比较 Go 版本号
- **iochan**: 在详细模式下处理工具链构建的输出流

## 注意事项

1. **Go 1.5+**: Go 1.5 及以上版本不需要构建工具链，Gox 可以直接使用
2. **CGO 支持**: 默认情况下，CGO 仅在同平台构建时启用
3. **Solaris 系统**: 在 Solaris 系统上，默认并行数限制为 3
4. **Windows 路径**: 处理 Windows 特有的路径格式转换
5. **模块支持**: 自动检测 Go 版本并启用 `-mod` 标志（Go 1.11+）

## 扩展开发

### 添加新功能

1. **新编译标志**: 在 `main.go` 中添加标志，在 `CompileOpts` 中添加字段
2. **新平台变量**: 在 `OutputTemplateData` 中添加字段，在模板中可用
3. **新环境变量**: 在 `envOverride` 函数中添加新的键类型

### 调试技巧

```bash
# 详细模式查看构建过程
gox -verbose -osarch="linux/amd64"

# 列出所有支持的平台
gox -osarch-list

# 测试特定平台构建
gox -osarch="linux/amd64" -output="/dev/null"
```

## 参考资源

- [GitHub 仓库](https://github.com/mitchellh/gox)
- [Go 交叉编译文档](https://golang.org/doc/install/source#environment)
- [Go 版本支持策略](https://golang.org/doc/devel/release.html#policy)

---

*最后更新: 2025年12月18日*  
*基于项目分析和 README 文档生成*