package api

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"
)

func TestConvertOutfitSpritePNGToGIF(t *testing.T) {
	sprite := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	red := color.NRGBA{R: 255, A: 255}
	blue := color.NRGBA{B: 255, A: 255}
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			sprite.Set(x, y, red)
		}
		for x := 2; x < 4; x++ {
			sprite.Set(x, y, blue)
		}
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, sprite); err != nil {
		t.Fatalf("encode png fixture: %v", err)
	}

	encoded, err := convertOutfitSpritePNGToGIF(pngBuf.Bytes())
	if err != nil {
		t.Fatalf("convert to gif: %v", err)
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("decode gif result: %v", err)
	}
	if len(decoded.Image) != 2 {
		t.Fatalf("expected 2 gif frames, got %d", len(decoded.Image))
	}
}

func TestMaybeFormatOutfitImageGIF(t *testing.T) {
	sprite := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	sprite.Set(0, 0, color.NRGBA{R: 255, A: 255})
	sprite.Set(1, 0, color.NRGBA{R: 255, A: 255})
	sprite.Set(0, 1, color.NRGBA{R: 255, A: 255})
	sprite.Set(1, 1, color.NRGBA{R: 255, A: 255})

	var pngBuf bytes.Buffer
	err := png.Encode(&pngBuf, sprite)
	if err != nil {
		t.Fatalf("encode png fixture: %v", err)
	}

	body, contentType, err := maybeFormatOutfitImage("gif", pngBuf.Bytes(), "image/png")
	if err != nil {
		t.Fatalf("format gif: %v", err)
	}
	if contentType != "image/gif" {
		t.Fatalf("expected image/gif, got %q", contentType)
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("decode returned gif: %v", err)
	}
	if len(decoded.Image) == 0 {
		t.Fatal("expected at least one gif frame")
	}
}
