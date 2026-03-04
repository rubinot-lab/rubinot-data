package api

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	"image/png"
	"strings"
)

const (
	outfitGIFDefaultFrameDelay = 8
	outfitGIFTwoFrameDelay     = 30
)

var (
	outfitGIFOpaquePalette = color.Palette(palette.Plan9[:255])
	outfitGIFPalette       = func() color.Palette {
		p := make(color.Palette, 1+len(outfitGIFOpaquePalette))
		p[0] = color.NRGBA{R: 0, G: 0, B: 0, A: 0}
		copy(p[1:], outfitGIFOpaquePalette)
		return p
	}()
)

func maybeFormatOutfitImage(format string, body []byte, contentType string) ([]byte, string, error) {
	requested := strings.ToLower(strings.TrimSpace(format))
	if requested != "gif" {
		if strings.TrimSpace(contentType) == "" {
			contentType = "image/png"
		}
		return body, contentType, nil
	}

	encoded, err := convertOutfitSpritePNGToGIF(body)
	if err != nil {
		return nil, "", err
	}
	return encoded, "image/gif", nil
}

func convertOutfitSpritePNGToGIF(body []byte) ([]byte, error) {
	sprite, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("decode outfit png: %w", err)
	}

	bounds := sprite.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid outfit sprite dimensions %dx%d", width, height)
	}

	frameWidth := width
	frameCount := 1
	if width%height == 0 && width > height {
		frameWidth = height
		frameCount = width / height
	}

	frames := make([]*image.Paletted, 0, frameCount)
	delays := make([]int, 0, frameCount)
	disposals := make([]byte, 0, frameCount)
	frameDelay := frameDelayForCount(frameCount)
	for i := 0; i < frameCount; i++ {
		paletted := image.NewPaletted(image.Rect(0, 0, frameWidth, height), outfitGIFPalette)
		srcPoint := image.Point{X: bounds.Min.X + (i * frameWidth), Y: bounds.Min.Y}
		drawFrameWithTransparency(paletted, sprite, srcPoint)
		frames = append(frames, paletted)
		delays = append(delays, frameDelay)
		disposals = append(disposals, gif.DisposalBackground)
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &gif.GIF{
		Image:           frames,
		Delay:           delays,
		Disposal:        disposals,
		BackgroundIndex: 0,
		LoopCount:       0,
	}); err != nil {
		return nil, fmt.Errorf("encode outfit gif: %w", err)
	}

	return buf.Bytes(), nil
}

func frameDelayForCount(frameCount int) int {
	if frameCount == 2 {
		return outfitGIFTwoFrameDelay
	}
	return outfitGIFDefaultFrameDelay
}

func drawFrameWithTransparency(dst *image.Paletted, src image.Image, srcPoint image.Point) {
	bounds := dst.Bounds()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			pixel := color.NRGBAModel.Convert(src.At(srcPoint.X+x, srcPoint.Y+y)).(color.NRGBA)
			if pixel.A < 128 {
				dst.SetColorIndex(x, y, 0)
				continue
			}
			dst.SetColorIndex(x, y, uint8(1+outfitGIFOpaquePalette.Index(pixel)))
		}
	}
}
