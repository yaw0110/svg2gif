package converter

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"sort"
)

// SVGAConverter handles SVGA to GIF conversion
type SVGAConverter struct{}

// NewSVGAConverter creates a new SVGA converter
func NewSVGAConverter() *SVGAConverter {
	return &SVGAConverter{}
}

// Format returns the SVGA format
func (c *SVGAConverter) Format() Format {
	return FormatSVGA
}

// Convert converts an SVGA file to GIF frames
func (c *SVGAConverter) Convert(r io.Reader, opts Options) ([]image.Image, []int, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read SVGA: %w", err)
	}

	movie, err := c.parseSVGA(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse SVGA: %w", err)
	}

	// Calculate actual frame count from sprite frame indices
	// The movie.Frames field may not be accurate
	actualFrames := 0
	for _, sprite := range movie.Sprites {
		for _, f := range sprite.Frames {
			frameIdx := int(f.FrameIndex)
			// Convert negative indices (in two's complement representation for large numbers)
			if frameIdx < 0 {
				frameIdx = -frameIdx
			}
			if frameIdx+1 > actualFrames {
				actualFrames = frameIdx + 1
			}
		}
	}

	// If sprites have more frames than movie declares, use that
	if actualFrames > int(movie.Frames) {
		movie.Frames = int32(actualFrames)
	}

	// If still no frames, default to number of images or 1
	if movie.Frames <= 0 {
		if len(movie.Images) > 0 {
			movie.Frames = int32(len(movie.Images))
		} else {
			movie.Frames = 1
		}
	}

	width := int(movie.Width)
	height := int(movie.Height)

	// Calculate scale factors for optional output dimensions
	scaleX := 1.0
	scaleY := 1.0
	if opts.Width > 0 && opts.Height > 0 {
		scaleX = float64(opts.Width) / float64(width)
		scaleY = float64(opts.Height) / float64(height)
		width = opts.Width
		height = opts.Height
	}

	fps := int(movie.Fps)
	if fps <= 0 {
		fps = opts.FPS
	}
	if fps <= 0 {
		fps = 20
	}

	// Decode images
	images := make(map[string]image.Image)
	imageSizes := make(map[string][2]int) // Store image dimensions for position mode detection
	for key, imgData := range movie.Images {
		img, err := c.decodeImage(imgData)
		if err != nil {
			fmt.Printf("Warning: failed to decode image %s: %v\n", key, err)
			continue
		}
		images[key] = img
		bounds := img.Bounds()
		imageSizes[key] = [2]int{bounds.Dx(), bounds.Dy()}
	}

	// Detect position mode if auto
	if opts.PositionMode == PositionModeAuto || opts.PositionMode == "" {
		movie.PositionMode = c.detectPositionMode(movie, imageSizes)
	} else {
		movie.PositionMode = opts.PositionMode
	}

	// If no sprites but has images, use images directly as frames
	if len(movie.Sprites) == 0 && len(movie.Images) > 0 {
		// Sort image keys
		keys := make([]string, 0, len(movie.Images))
		for k := range movie.Images {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		frameCount := len(keys)
		if frameCount > actualFrames {
			frameCount = actualFrames
		}

		frames := make([]image.Image, frameCount)
		delays := make([]int, frameCount)

		for i := 0; i < frameCount; i++ {
			frame := image.NewRGBA(image.Rect(0, 0, width, height))
			// Fill with transparent
			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					frame.Set(x, y, color.RGBA{0, 0, 0, 0})
				}
			}

			// Draw image centered
			if i < len(keys) && images[keys[i]] != nil {
				c.drawImageCentered(frame, images[keys[i]], width, height)
			}
			frames[i] = frame
			delays[i] = 100 / fps
		}
		return frames, delays, nil
	}

	frames := make([]image.Image, actualFrames)
	delays := make([]int, actualFrames)

	// Render each frame
	for i := 0; i < actualFrames; i++ {
		frame := c.renderFrame(movie, images, i, width, height, scaleX, scaleY)
		frames[i] = frame
		delays[i] = 100 / fps
	}

	return frames, delays, nil
}

// detectPositionMode analyzes SVGA data to determine how positions should be interpreted
func (c *SVGAConverter) detectPositionMode(movie *Movie, imageSizes map[string][2]int) PositionMode {
	canvasW, canvasH := float64(movie.Width), float64(movie.Height)

	// Find most common image size
	imageSizeCount := make(map[[2]int]int)
	for _, size := range imageSizes {
		imageSizeCount[size]++
	}

	var commonW, commonH int
	maxCount := 0
	for size, count := range imageSizeCount {
		if count > maxCount {
			maxCount = count
			commonW = size[0]
			commonH = size[1]
		}
	}

	// Collect all non-zero positions
	positions := make(map[[2]float32]int)
	for _, sprite := range movie.Sprites {
		for _, f := range sprite.Frames {
			if f.X != 0 || f.Y != 0 {
				positions[[2]float32{f.X, f.Y}]++
			}
		}
	}

	// Check position patterns
	for pos := range positions {
		px, py := float64(pos[0]), float64(pos[1])

		// Mode 1: position equals canvas size (center)
		if px == canvasW && py == canvasH {
			return PositionModeCanvasSize
		}

		// Mode 2: position equals image size (center)
		if px == float64(commonW) && py == float64(commonH) {
			return PositionModeImageSize
		}

		// Mode 3: position is absolute center
		if px == canvasW/2 && py == canvasH/2 {
			return PositionModeCenter
		}
	}

	// Default: position is sprite center
	return PositionModeAbsolute
}

// ProtobufReader helps read protobuf wire format
type ProtobufReader struct {
	data []byte
	pos  int
}

func (r *ProtobufReader) readVarint() uint64 {
	var result uint64
	var shift uint
	for r.pos < len(r.data) {
		b := r.data[r.pos]
		r.pos++
		result |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result
}

func (r *ProtobufReader) readFixed32() float32 {
	if r.pos+4 > len(r.data) {
		return 0
	}
	bits := binary.LittleEndian.Uint32(r.data[r.pos : r.pos+4])
	r.pos += 4
	return math.Float32frombits(bits)
}

func (r *ProtobufReader) readBytes() []byte {
	length := r.readVarint()
	if r.pos+int(length) > len(r.data) {
		return nil
	}
	data := r.data[r.pos : r.pos+int(length)]
	r.pos += int(length)
	return data
}

func (r *ProtobufReader) remaining() int {
	return len(r.data) - r.pos
}

// parseSVGA parses SVGA protobuf format
func (c *SVGAConverter) parseSVGA(data []byte) (*Movie, error) {
	// Decompress zlib
	if len(data) >= 2 && data[0] == 0x78 {
		r, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create zlib reader: %w", err)
		}
		defer r.Close()
		data, err = io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress: %w", err)
		}
	}

	movie := &Movie{
		Images:  make(map[string][]byte),
		Sprites: make([]*Sprite, 0),
	}

	reader := &ProtobufReader{data: data}

	for reader.remaining() > 0 {
		tag := reader.readVarint()
		field := tag >> 3
		wire := tag & 0x7

		switch field {
		case 1: // version
			if wire == 2 {
				movie.Version = string(reader.readBytes())
			}
		case 2: // params
			if wire == 2 {
				paramsData := reader.readBytes()
				c.parseParams(paramsData, movie)
			}
		case 3: // images map entry
			if wire == 2 {
				entryData := reader.readBytes()
				key, value := c.parseMapEntry(entryData)
				if key != "" && len(value) > 0 {
					movie.Images[key] = value
				}
			}
		case 4: // sprites
			if wire == 2 {
				spriteData := reader.readBytes()
				sprite := c.parseSprite(spriteData)
				if sprite != nil {
					movie.Sprites = append(movie.Sprites, sprite)
				}
			}
		default:
			if wire == 0 {
				reader.readVarint()
			} else if wire == 2 {
				reader.readBytes()
			} else if wire == 5 {
				reader.pos += 4
			}
		}
	}

	return movie, nil
}

func (c *SVGAConverter) parseParams(data []byte, movie *Movie) {
	reader := &ProtobufReader{data: data}

	for reader.remaining() > 0 {
		tag := reader.readVarint()
		field := tag >> 3
		wire := tag & 0x7

		switch field {
		case 1: // width
			if wire == 5 {
				movie.Width = int32(reader.readFixed32())
			}
		case 2: // height
			if wire == 5 {
				movie.Height = int32(reader.readFixed32())
			}
		case 3: // frames
			if wire == 0 {
				movie.Frames = int32(reader.readVarint())
			}
		case 4: // fps
			if wire == 0 {
				movie.Fps = int32(reader.readVarint())
			}
		default:
			if wire == 0 {
				reader.readVarint()
			} else if wire == 2 {
				reader.readBytes()
			} else if wire == 5 {
				reader.pos += 4
			}
		}
	}
}

func (c *SVGAConverter) parseMapEntry(data []byte) (string, []byte) {
	reader := &ProtobufReader{data: data}
	var key string
	var value []byte

	for reader.remaining() > 0 {
		tag := reader.readVarint()
		field := tag >> 3
		wire := tag & 0x7

		if field == 1 && wire == 2 {
			key = string(reader.readBytes())
		} else if field == 2 && wire == 2 {
			value = reader.readBytes()
		} else {
			if wire == 0 {
				reader.readVarint()
			} else if wire == 2 {
				reader.readBytes()
			}
		}
	}

	return key, value
}

func (c *SVGAConverter) parseSprite(data []byte) *Sprite {
	sprite := &Sprite{Frames: make([]*Frame, 0)}
	reader := &ProtobufReader{data: data}
	frameIndex := int32(0)

	for reader.remaining() > 0 {
		tag := reader.readVarint()
		field := tag >> 3
		wire := tag & 0x7

		switch field {
		case 1: // imageKey
			if wire == 2 {
				sprite.ImageKey = string(reader.readBytes())
			}
		case 2: // frames array - each entry is one frame, index is implicit
			if wire == 2 {
				frameData := reader.readBytes()
				frame := c.parseFrame(frameData)
				if frame != nil {
					frame.FrameIndex = frameIndex
					sprite.Frames = append(sprite.Frames, frame)
				}
				frameIndex++
			}
		default:
			if wire == 0 {
				reader.readVarint()
			} else if wire == 2 {
				reader.readBytes()
			}
		}
	}

	return sprite
}

func (c *SVGAConverter) parseFrame(data []byte) *Frame {
	frame := &Frame{Alpha: 1, ScaleX: 1, ScaleY: 1}
	reader := &ProtobufReader{data: data}

	for reader.remaining() > 0 {
		tag := reader.readVarint()
		field := tag >> 3
		wire := tag & 0x7

		switch field {
		case 1: // alpha
			if wire == 5 {
				frame.Alpha = reader.readFixed32()
			}
		case 2: // Transform (embedded message)
			if wire == 2 {
				transformData := reader.readBytes()
				c.parseTransform(transformData, frame)
			}
		default:
			if wire == 0 {
				reader.readVarint()
			} else if wire == 2 {
				reader.readBytes()
			} else if wire == 5 {
				reader.pos += 4
			}
		}
	}

	return frame
}

func (c *SVGAConverter) parseTransform(data []byte, frame *Frame) {
	reader := &ProtobufReader{data: data}

	for reader.remaining() > 0 {
		tag := reader.readVarint()
		field := tag >> 3
		wire := tag & 0x7

		switch field {
		case 3: // x
			if wire == 5 {
				frame.X = reader.readFixed32()
			}
		case 4: // y
			if wire == 5 {
				frame.Y = reader.readFixed32()
			}
		case 5: // scaleX
			if wire == 5 {
				frame.ScaleX = reader.readFixed32()
			}
		case 6: // scaleY
			if wire == 5 {
				frame.ScaleY = reader.readFixed32()
			}
		case 7: // rotation
			if wire == 5 {
				frame.Rotation = reader.readFixed32()
			}
		default:
			if wire == 0 {
				reader.readVarint()
			} else if wire == 2 {
				reader.readBytes()
			} else if wire == 5 {
				reader.pos += 4
			}
		}
	}
}

func (c *SVGAConverter) decodeImage(data []byte) (image.Image, error) {
	if len(data) > 8 && bytes.HasPrefix(data, []byte("\x89PNG")) {
		return png.Decode(bytes.NewReader(data))
	}
	return nil, fmt.Errorf("not a PNG image")
}

// renderFrame renders a single animation frame
func (c *SVGAConverter) renderFrame(movie *Movie, images map[string]image.Image, frameIndex, width, height int, scaleX, scaleY float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Transparent background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	// Find sprites that should be visible at this frame
	// A sprite is visible if its position is not at origin (0,0)
	// Position equal to canvas dimensions means centered and visible
	for _, sprite := range movie.Sprites {
		var frame *Frame

		// Find frame data for current index
		for _, f := range sprite.Frames {
			if int(f.FrameIndex) == frameIndex {
				frame = f
				break
			}
		}

		// Get the image
		spriteImg, ok := images[sprite.ImageKey]
		if !ok {
			continue
		}

		// If frame has valid position data (not at origin), draw it
		// Position at origin (0,0) means hidden/off-screen
		// Position equal to canvas dimensions means centered
		if frame != nil && (frame.X != 0 || frame.Y != 0) {
			c.drawSpriteAtPosition(img, spriteImg, frame, width, height, scaleX, scaleY, int(movie.Width), int(movie.Height), movie.PositionMode)
		}
		// Sprites with position (0,0) are hidden - don't draw them
	}

	return img
}

// drawSpriteAtPosition draws sprite at the position specified in frame data
func (c *SVGAConverter) drawSpriteAtPosition(dst *image.RGBA, src image.Image, frame *Frame, width, height int, scaleX, scaleY float64, origWidth, origHeight int, mode PositionMode) {
	srcBounds := src.Bounds()
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()

	// Apply scaling to source dimensions
	scaledW := int(float64(srcW) * scaleX)
	scaledH := int(float64(srcH) * scaleY)

	// Skip hidden sprites
	if frame.X == 0 && frame.Y == 0 {
		return
	}

	// Calculate position based on mode
	var centerX, centerY float64

	switch mode {
	case PositionModeCanvasSize:
		// Position equals canvas size means centered
		centerX = float64(width) / 2
		centerY = float64(height) / 2
	case PositionModeImageSize:
		// Position equals image size means centered
		centerX = float64(width) / 2
		centerY = float64(height) / 2
	case PositionModeCenter:
		// Position is already the center
		centerX = float64(frame.X) * scaleX
		centerY = float64(frame.Y) * scaleY
	default:
		// PositionModeAbsolute: position is sprite center
		centerX = float64(frame.X) * scaleX
		centerY = float64(frame.Y) * scaleY
	}

	// Calculate top-left corner position
	startX := int(centerX - float64(scaledW)/2)
	startY := int(centerY - float64(scaledH)/2)

	// Apply alpha if needed
	alpha := float64(frame.Alpha)
	if alpha <= 0 {
		alpha = 1
	}

	// Draw scaled sprite using bilinear interpolation for better quality
	for y := 0; y < scaledH; y++ {
		for x := 0; x < scaledW; x++ {
			// Bilinear interpolation coordinates
			srcXF := float64(x) / scaleX
			srcYF := float64(y) / scaleY

			// Get integer and fractional parts
			srcX0 := int(srcXF)
			srcY0 := int(srcYF)
			srcX1 := srcX0 + 1
			srcY1 := srcY0 + 1

			// Clamp to source bounds
			if srcX0 >= srcW {
				srcX0 = srcW - 1
			}
			if srcY0 >= srcH {
				srcY0 = srcH - 1
			}
			if srcX1 >= srcW {
				srcX1 = srcW - 1
			}
			if srcY1 >= srcH {
				srcY1 = srcH - 1
			}

			// Fractional parts for interpolation
			fx := srcXF - float64(srcX0)
			fy := srcYF - float64(srcY0)

			// Get four neighboring pixels
			c00 := src.At(srcX0+srcBounds.Min.X, srcY0+srcBounds.Min.Y)
			c10 := src.At(srcX1+srcBounds.Min.X, srcY0+srcBounds.Min.Y)
			c01 := src.At(srcX0+srcBounds.Min.X, srcY1+srcBounds.Min.Y)
			c11 := src.At(srcX1+srcBounds.Min.X, srcY1+srcBounds.Min.Y)

			// Interpolate
			r00, g00, b00, a00 := c00.RGBA()
			r10, g10, b10, a10 := c10.RGBA()
			r01, g01, b01, a01 := c01.RGBA()
			r11, g11, b11, a11 := c11.RGBA()

			// Bilinear interpolation
			r := bilinear(r00, r10, r01, r11, fx, fy)
			g := bilinear(g00, g10, g01, g11, fx, fy)
			b := bilinear(b00, b10, b01, b11, fx, fy)
			a := bilinear(a00, a10, a01, a11, fx, fy)

			// Apply alpha
			if alpha < 1 {
				a = uint32(float64(a) * alpha)
			}

			dstX := startX + x
			dstY := startY + y

			// Allow drawing outside canvas bounds - just clip
			if dstX < 0 || dstX >= width || dstY < 0 || dstY >= height {
				continue
			}

			if a > 0 {
				dst.Set(dstX, dstY, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: uint8(a >> 8),
				})
			}
		}
	}
}

// bilinear performs bilinear interpolation between four values
func bilinear(v00, v10, v01, v11 uint32, fx, fy float64) uint32 {
	v0 := uint32(float64(v00) * (1-fx) + float64(v10) * fx)
	v1 := uint32(float64(v01) * (1-fx) + float64(v11) * fx)
	return uint32(float64(v0) * (1-fy) + float64(v1) * fy)
}

// drawImageCentered draws an image centered on the canvas
func (c *SVGAConverter) drawImageCentered(dst *image.RGBA, src image.Image, width, height int) {
	srcBounds := src.Bounds()
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()

	startX := (width - srcW) / 2
	startY := (height - srcH) / 2

	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			dstX := startX + x
			dstY := startY + y

			if dstX >= 0 && dstX < width && dstY >= 0 && dstY < height {
				srcColor := src.At(x+srcBounds.Min.X, y+srcBounds.Min.Y)
				dst.Set(dstX, dstY, srcColor)
			}
		}
	}
}

// Movie represents SVGA movie data
type Movie struct {
	Version      string
	Fps          int32
	Frames       int32
	Width        int32
	Height       int32
	Images       map[string][]byte
	Sprites      []*Sprite
	PositionMode PositionMode // Detected or configured position mode
}

// Sprite represents a sprite element
type Sprite struct {
	ImageKey string
	Frames   []*Frame
}

// Frame represents frame transformation data
type Frame struct {
	FrameIndex int32
	Alpha      float32
	ScaleX     float32
	ScaleY     float32
	Rotation   float32
	X          float32
	Y          float32
}
