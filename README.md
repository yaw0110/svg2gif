# svg2gif

SVG/SVGA 批量转换为 GIF 工具

## 功能特性

- 支持 SVG 和 SVGA 格式转换为 GIF
- **内置 ffmpeg**: 无需安装额外依赖，开箱即用
- **智能位置检测**：自动识别 SVGA 精灵位置模式
- 批量转换，自动匹配文件名
- 高质量输出，使用双线性插值算法
- 支持自定义尺寸和帧率
- 跨平台支持：Windows、macOS、Linux

## 快速开始

### Windows

1. 解压 `svg2gif-windows-amd64.zip`
2. 将 SVG/SVGA 文件放入 `source` 目录
3. 双击 `svg2gif.bat` 运行
4. GIF 文件输出到 `target/gif` 目录，APNG 文件输出到 `target/apng` 目录

### macOS / Linux

```bash
# 解压下载的 zip 文件
unzip svg2gif-mac-arm64.zip  # Apple Silicon
unzip svg2gif-mac-amd64.zip  # Intel Mac
unzip svg2gif-linux-amd64.zip

# 赋予执行权限
chmod +x svg2gif ffmpeg

# 运行转换
./svg2gif source target
```

## 命令行参数

```
Usage: svg2gif [options] <source> <target>

Options:
  -w, --width <pixels>       输出宽度 (默认: 自动)
  -h, --height <pixels>      输出高度 (默认: 自动)
  -f, --fps <number>         帧率 FPS (默认: 20)
  --help                     显示帮助
  --version                  显示版本
```

## 使用示例

```bash
# 基本使用
./svg2gif source target

# 自定义尺寸
./svg2gif -w 800 --height 800 source target

# 自定义帧率
./svg2gif --fps 24 source target
```

## 目录结构

```
svg2gif/
├── cmd/
│   └── svg2gif/
│       └── main.go          # 程序入口
├── pkg/
│   ├── converter/
│   │   ├── converter.go     # 转换器接口
│   │   ├── svg.go           # SVG 转换实现
│   │   └── svga.go          # SVGA 转换实现
│   └── apng/
│       └── encoder.go       # APNG 编码器
├── dist/                    # 编译输出目录
├── source/                  # 源文件目录
├── target/                  # 输出目录
├── go.mod
├── go.sum
└── README.md
```

## 转换流程

```
source/              target/
├── img1.svga   →    ├── apng/
├── img2.svga   →    │   ├── img1.png
├── img3.svg    →    │   ├── img2.png
                     │   └── img3.png
                     └── gif/
                         ├── img1.gif
                         ├── img2.gif
                         └── img3.gif
```

## 技术实现

- **SVG 解析**: 使用 `oksvg` 库解析和渲染 SVG
- **SVGA 解析**: 使用 Protocol Buffers 解析 SVGA 格式
- **位置检测**: 自动分析精灵位置模式，智能居中处理
- **图像缩放**: CatmullRom 插值算法，保证输出质量
- **APNG 编码**: 使用 `kettek/apng` 库
- **GIF 转换**: 使用 ffmpeg 进行 APNG 到 GIF 的转换

## 编译

```bash
# 编译当前平台
go build -o svg2gif ./cmd/svg2gif

# 编译所有平台
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/svg2gif-mac-amd64 ./cmd/svg2gif
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/svg2gif-mac-arm64 ./cmd/svg2gif
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/svg2gif-linux-amd64 ./cmd/svg2gif
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/svg2gif-windows-amd64.exe ./cmd/svg2gif
```

## 依赖

- Go 1.21+
- github.com/srwiley/oksvg
- github.com/srwiley/rasterx
- github.com/kettek/apng

## 版本历史

### v1.4.0
- 内置 ffmpeg，无需额外安装
- 输出 APNG 文件到 `target/apng` 目录
- 输出 GIF 文件到 `target/gif` 目录
- 自动检测同目录下的 ffmpeg 可执行文件

### v1.3.0
- 新增智能位置检测，自动识别 SVGA 精灵位置模式
- 支持手动配置位置模式
- 修复精灵位置偏移问题

### v1.2.0
- 新增批量转换功能
- 优化图像缩放质量（双线性插值）
- 新增 Windows 启动脚本

### v1.1.0
- 支持 SVGA 格式转换
- 支持自定义输出尺寸

### v1.0.0
- 初始版本
- 支持 SVG 转 GIF

## License

MIT License
