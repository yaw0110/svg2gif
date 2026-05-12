package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhyan/svg2gif/pkg/converter"
	gifencoder "github.com/jhyan/svg2gif/pkg/gif"
)

var (
	version = "1.2.0"
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
	var inputDir, outputDir string
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
			if inputDir == "" {
				inputDir = args[i]
			} else if outputDir == "" {
				outputDir = args[i]
			}
			i++
		}
	}

	if inputDir == "" || outputDir == "" {
		fmt.Fprintln(os.Stderr, "Error: input and output directories are required")
		printUsage()
		os.Exit(1)
	}

	// 检查输入目录
	info, err := os.Stat(inputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: input directory not found: %s\n", inputDir)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", inputDir)
		os.Exit(1)
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// 扫描输入目录
	files, err := filepath.Glob(filepath.Join(inputDir, "*"))
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
	fmt.Printf("Input:  %s\n", inputDir)
	fmt.Printf("Output: %s\n", outputDir)
	fmt.Println()

	// 批量转换
	success, failed := 0, 0
	startTime := time.Now()

	for i, inputFile := range validFiles {
		filename := filepath.Base(inputFile)
		ext := filepath.Ext(filename)
		baseName := strings.TrimSuffix(filename, ext)
		outputFile := filepath.Join(outputDir, baseName+".gif")

		fmt.Printf("[%d/%d] Converting: %s\n", i+1, len(validFiles), filename)

		err := convertFile(inputFile, outputFile, width, height, fps)
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

func convertFile(inputPath, outputPath string, width, height, fps int) error {
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

	imgWidth, imgHeight := width, height
	if imgWidth <= 0 || imgHeight <= 0 {
		if len(frames) > 0 {
			bounds := frames[0].Bounds()
			imgWidth = bounds.Dx()
			imgHeight = bounds.Dy()
		} else {
			imgWidth, imgHeight = 800, 600
		}
	}

	encoder := gifencoder.NewEncoder(imgWidth, imgHeight, fps)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}
	defer outputFile.Close()

	if len(frames) == 1 {
		err = encoder.EncodeStatic(outputFile, frames[0])
	} else {
		err = encoder.EncodeFrames(outputFile, frames, delays)
	}

	if err != nil {
		return fmt.Errorf("encoding failed: %w", err)
	}

	fileInfo, _ := os.Stat(outputPath)
	fmt.Printf("  -> %s (%.1f KB)\n", filepath.Base(outputPath), float64(fileInfo.Size())/1024)

	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `svg2gif - Batch convert SVG/SVGA to GIF

Usage: svg2gif [options] <input_dir> <output_dir>

Options:
  -w, --width <pixels>   Output width (default: auto)
  -h, --height <pixels>  Output height (default: auto)
  -f, --fps <number>     Frames per second (default: 20)
  --help                 Show this help
  --version              Show version

Examples:
  svg2gif ./svga ./output
  svg2gif -w 800 -h 800 ./svga ./output
  svg2gif --fps 24 ./svga ./output

`)
}

// 保留单文件转换支持
func init() {
	// 如果想支持单文件模式，可以在这里检测参数
}
