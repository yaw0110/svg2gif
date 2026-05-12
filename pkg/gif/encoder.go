// Package gif provides GIF encoding utilities for creating animated GIFs
package gif

import (
	"image"
	"image/color/palette"
	"image/gif"
	"io"
	"time"
)

// Encoder handles GIF animation encoding
type Encoder struct {
	Width, Height int
	FPS           int
	Delay         int // Delay between frames in 100ths of a second
}

// NewEncoder creates a new GIF encoder with the given parameters
func NewEncoder(width, height, fps int) *Encoder {
	if fps <= 0 {
		fps = 15
	}
	if width <= 0 {
		width = 800
	}
	if height <= 0 {
		height = 600
	}
	return &Encoder{
		Width:  width,
		Height: height,
		FPS:    fps,
		Delay:  100 / fps, // GIF delay is in 100ths of a second
	}
}

// EncodeFrames encodes a slice of images as an animated GIF
func (e *Encoder) EncodeFrames(w io.Writer, frames []image.Image, delays []int) error {
	if len(frames) == 0 {
		return nil
	}

	out := &gif.GIF{}

	// Use consistent delays if not provided
	if len(delays) == 0 {
		delays = make([]int, len(frames))
		for i := range delays {
			delays[i] = e.Delay
		}
	}

	// Convert frames to paletted images with better quality
	pal := palette.Plan9
	for i, frame := range frames {
		// Use frame dimensions directly - no additional resize needed
		// The converter should handle sizing properly

		// Convert to paletted image with better color matching
		paletted := image.NewPaletted(frame.Bounds(), pal)
		bounds := frame.Bounds()

		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				// Use better color matching - find closest palette color
				c := frame.At(x, y)
				paletted.Set(x-bounds.Min.X, y-bounds.Min.Y, c)
			}
		}

		out.Image = append(out.Image, paletted)
		if i < len(delays) {
			out.Delay = append(out.Delay, delays[i])
		} else {
			out.Delay = append(out.Delay, e.Delay)
		}
	}

	return gif.EncodeAll(w, out)
}

// EncodeStatic encodes a single static image as a GIF with better quality
func (e *Encoder) EncodeStatic(w io.Writer, img image.Image) error {
	// Use better encoding options
	options := &gif.Options{
		NumColors: 256,
	}
	return gif.Encode(w, img, options)
}

// Duration calculates the total duration of the GIF animation
func (e *Encoder) Duration(frameCount int) time.Duration {
	return time.Duration(frameCount*e.Delay*10) * time.Millisecond
}