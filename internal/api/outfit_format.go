package api

import (
	"bytes"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"strings"
)

const outfitGIFFrameDelay = 8

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
	for i := 0; i < frameCount; i++ {
		paletted := image.NewPaletted(image.Rect(0, 0, frameWidth, height), palette.Plan9)
		srcPoint := image.Point{X: bounds.Min.X + (i * frameWidth), Y: bounds.Min.Y}
		draw.FloydSteinberg.Draw(paletted, paletted.Rect, sprite, srcPoint)
		frames = append(frames, paletted)
		delays = append(delays, outfitGIFFrameDelay)
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &gif.GIF{
		Image:     frames,
		Delay:     delays,
		LoopCount: 0,
	}); err != nil {
		return nil, fmt.Errorf("encode outfit gif: %w", err)
	}

	return buf.Bytes(), nil
}
