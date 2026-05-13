package main

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/gif"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhyan/svg2gif/pkg/converter"
)

var (
	version = "1.5.0"
)

func main() {
	args := os.Args[1:]

	// 显示帮助
	if len(args) < 1 || args[0] == "--help" || args[0] == "-h" {
		printUsage()
		os.Exit(0)
	}

	if args[0] == "--version" || args[0] == "-v" {
		fmt.Printf("svg2gif version %s\n", version)
		os.Exit(0)
	}

	// 解析参数
	var source, target string
	width, height, fps := 0, 0, 20

	i := 0
	for i < len(args) {
		switch args[i] {
		case "--width", "-w":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &width)
				i += 2
			} else {
				fmt.Fprintln(os.Stderr, "Error: --width requires a value")
				os.Exit(1)
			}
		case "--height":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &height)
				i += 2
			} else {
				fmt.Fprintln(os.Stderr, "Error: --height requires a value")
				os.Exit(1)
			}
		case "--fps", "-f":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &fps)
				i += 2
			} else {
				fmt.Fprintln(os.Stderr, "Error: --fps requires a value")
				os.Exit(1)
			}
		default:
			if source == "" {
				source = args[i]
			} else if target == "" {
				target = args[i]
			}
			i++
		}
	}

	if source == "" || target == "" {
		fmt.Fprintln(os.Stderr, "Error: source and target are required")
		printUsage()
		os.Exit(1)
	}

	// 检查 source
	info, err := os.Stat(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: source not found: %s\n", source)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", source)
		os.Exit(1)
	}

	// 创建 target 目录
	if err := os.MkdirAll(target, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create target directory: %v\n", err)
		os.Exit(1)
	}

	// 扫描 source 目录
	files, err := filepath.Glob(filepath.Join(source, "*"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to scan input directory: %v\n", err)
		os.Exit(1)
	}

	// 过滤支持的文件格式
	var validFiles []string
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == ".svg" || ext == ".svga" {
			validFiles = append(validFiles, f)
		}
	}

	if len(validFiles) == 0 {
		fmt.Println("No SVG or SVGA files found in input directory")
		os.Exit(0)
	}

	fmt.Printf("Found %d file(s) to convert\n", len(validFiles))
	fmt.Printf("Source: %s\n", source)
	fmt.Printf("Target: %s\n", target)
	fmt.Println()

	// 批量转换
	success, failed := 0, 0
	startTime := time.Now()

	for i, inputFile := range validFiles {
		filename := filepath.Base(inputFile)
		ext := filepath.Ext(filename)
		baseName := strings.TrimSuffix(filename, ext)

		fmt.Printf("[%d/%d] Converting: %s\n", i+1, len(validFiles), filename)

		gifOutput := filepath.Join(target, baseName+".gif")

		err := convertFile(inputFile, gifOutput, width, height, fps)
		if err != nil {
			fmt.Printf("  Failed: %v\n", err)
			failed++
		} else {
			success++
		}
	}

	elapsed := time.Since(startTime)
	fmt.Println()
	fmt.Println("========== Summary ==========")
	fmt.Printf("Total:   %d file(s)\n", len(validFiles))
	fmt.Printf("Success: %d\n", success)
	fmt.Printf("Failed:  %d\n", failed)
	fmt.Printf("Time:    %.2f seconds\n", elapsed.Seconds())
}

func convertFile(inputPath, gifOutput string, width, height, fps int) error {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer inputFile.Close()

	format := converter.DetectFormat(inputPath)

	var conv converter.Converter
	switch format {
	case converter.FormatSVGA:
		conv = converter.NewSVGAConverter()
	default:
		conv = converter.NewSVGConverter()
	}

	opts := converter.Options{
		Width:  width,
		Height: height,
		FPS:    fps,
	}

	frames, delays, err := conv.Convert(inputFile, opts)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	if len(frames) == 0 {
		return fmt.Errorf("no frames generated")
	}

	// 直接生成 GIF
	gifFile, err := os.Create(gifOutput)
	if err != nil {
		return fmt.Errorf("failed to create GIF: %w", err)
	}
	defer gifFile.Close()

	err = encodeGIF(gifFile, frames, delays)
	if err != nil {
		return fmt.Errorf("GIF encoding failed: %w", err)
	}

	gifInfo, _ := os.Stat(gifOutput)
	fmt.Printf("  GIF:  %s (%.1f KB)\n", filepath.Base(gifOutput), float64(gifInfo.Size())/1024)

	return nil
}

// encodeGIF 编码帧为 GIF
func encodeGIF(w *os.File, frames []image.Image, delays []int) error {
	if len(frames) == 0 {
		return nil
	}

	// 使用 Plan9 调色板
	pal := palette.Plan9

	g := &gif.GIF{
		Image: make([]*image.Paletted, len(frames)),
		Delay: make([]int, len(frames)),
	}

	for i, frame := range frames {
		// 转换为调色板图像
		bounds := frame.Bounds()
		paletted := image.NewPaletted(bounds, pal)

		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				paletted.Set(x, y, frame.At(x, y))
			}
		}

		g.Image[i] = paletted
		// GIF delay 单位是 10ms，delays 是 100ms 为单位的帧间隔
		// delays[i] = 100/fps，所以 delay = delays[i] * 10
		g.Delay[i] = delays[i] * 10
	}

	return gif.EncodeAll(w, g)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `svg2gif - Batch convert SVG/SVGA to GIF

Usage: svg2gif [options] <source> <target>

Options:
  -w, --width <pixels>      Output width (default: auto)
  -h, --height <pixels>     Output height (default: auto)
  -f, --fps <number>        Frames per second (default: 20)
  --help                    Show this help
  --version                 Show version

Examples:
  svg2gif ./source ./target
  svg2gif -w 800 -h 800 ./source ./target
  svg2gif --fps 24 ./source ./target

`)
}
