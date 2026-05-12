// Package converter provides format conversion interfaces
package converter

import (
	"image"
	"io"
)

// Format represents the input file format
type Format string

const (
	FormatSVG  Format = "svg"
	FormatSVGA Format = "svga"
)

// PositionMode defines how to interpret sprite positions
type PositionMode string

const (
	// PositionModeAuto - automatically detect from SVGA data
	PositionModeAuto PositionMode = "auto"
	// PositionModeCanvasSize - position equals canvas size means center
	PositionModeCanvasSize PositionMode = "canvas_size"
	// PositionModeImageSize - position equals image size means center
	PositionModeImageSize PositionMode = "image_size"
	// PositionModeCenter - position is absolute center coordinates
	PositionModeCenter PositionMode = "center"
	// PositionModeAbsolute - position is sprite center (no centering)
	PositionModeAbsolute PositionMode = "absolute"
)

// Options contains conversion parameters
type Options struct {
	Width       int
	Height      int
	FPS         int
	PositionMode PositionMode // How to interpret sprite positions
}

// Converter defines the interface for format converters
type Converter interface {
	// Convert converts the input file to GIF frames
	Convert(r io.Reader, opts Options) ([]image.Image, []int, error)

	// Format returns the supported input format
	Format() Format
}

// DetectFormat detects the input format from the file extension or content
func DetectFormat(filename string) Format {
	if len(filename) >= 5 && filename[len(filename)-5:] == ".svga" {
		return FormatSVGA
	}
	return FormatSVG
}
