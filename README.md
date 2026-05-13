# svg2gif

SVG/SVGA 批量转换为 GIF 工具

## 功能特性

- 支持 SVG 和 SVGA 格式转换为 GIF
- 采用两步转换策略：SVGA → APNG → GIF，确保高质量输出
- 批量转换，自动匹配文件名
- 同时输出 APNG 和 GIF 文件，便于调试和对比
- 高质量渲染，使用 CatmullRom 插值算法
- 正确处理调色板 PNG 透明度问题
- 支持自定义尺寸和帧率
- 跨平台支持：Windows、macOS、Linux

## 快速开始

### Windows

1. 解压 `svg2gif-win.zip`
2. 将 SVG/SVGA 文件放入 `source` 目录
3. 双击 `svg2gif.bat` 运行
4. GIF 文件输出到 `target/gif` 目录，APNG 输出到 `target/apng` 目录

### macOS / Linux

```bash
# 赋予执行权限
chmod +x svg2gif-mac-arm64  # Apple Silicon
chmod +x svg2gif-mac-amd64  # Intel Mac
chmod +x svg2gif-linux-amd64

# 运行转换
./svg2gif-mac-arm64 source target
```

## 命令行参数

```
Usage: svg2gif [options] <source> <target>

Options:
  -w, --width <pixels>      输出宽度 (默认: 自动)
  -h, --height <pixels>     输出高度 (默认: 自动)
  -f, --fps <number>        帧率 FPS (默认: 20)
  --help                    显示帮助
  --version                 显示版本

Output:
  target/apng/  - APNG 文件 (调试用)
  target/gif/   - GIF 文件 (最终输出)
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
│   ├── apng/                # APNG 文件
│   └── gif/                 # GIF 文件
├── go.mod
├── go.sum
└── README.md
```

## 转换流程

```
source/                  target/
├── img1.svga   →        ├── apng/img1.png
├── img2.svga   →        │   ├── apng/img2.png
└── img3.svg    →        │   ├── apng/img3.png
                         └── gif/img1.gif
                             ├── gif/img2.gif
                             └── gif/img3.gif
```

## 技术实现

### SVGA 解析

- 解析 zlib 压缩的 Protocol Buffers 数据
- 正确处理稀疏帧数据（精灵只在特定帧出现）
- 解析 2D 仿射变换矩阵，支持缩放、旋转、倾斜
- 处理调色板 PNG 的透明度问题（alpha=0 时清零 RGB）

### 图像渲染

- 使用 CatmullRom 高质量插值算法进行图像缩放
- Porter-Duff source-over alpha 混合
- 预乘 alpha 双线性插值（Canvas 2D 标准方式）

### APNG → GIF

- 使用 ffmpeg 进行 APNG 到 GIF 转换
- palettegen + paletteuse 滤镜优化颜色质量

## 依赖

- Go 1.21+
- ffmpeg (用于 APNG → GIF 转换)
- github.com/srwiley/oksvg
- github.com/srwiley/rasterx
- github.com/kettek/apng
- golang.org/x/image/draw

## 编译

```bash
# 编译当前平台
go build -o svg2gif ./cmd/svg2gif

# 编译所有平台
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/svg2gif-mac-amd64 ./cmd/svg2gif
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/svg2gif-mac-arm64 ./cmd/svg2gif
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/svg2gif-linux-amd64 ./cmd/svg2gif
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/svg2gif-windows-amd64.exe ./cmd/svg2gif

# 打包
cd dist && zip svg2gif-mac-arm64.zip svg2gif-mac-arm64
cd dist && zip svg2gif-mac-amd64.zip svg2gif-mac-amd64
cd dist && zip svg2gif-linux-amd64.zip svg2gif-linux-amd64
cd dist && zip svg2gif-windows-amd64.zip svg2gif-windows-amd64.exe
```

## 版本历史

### v1.4.0
- 采用 SVGA → APNG → GIF 两步转换策略
- 同时输出 APNG 和 GIF 文件便于调试
- 修复帧数计算，正确处理稀疏帧数据
- 修复调色板 PNG 透明度问题
- 使用 CatmullRom 高质量插值算法
- 正确实现 Porter-Duff alpha 混合

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