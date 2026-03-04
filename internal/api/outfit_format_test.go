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
	for i, delay := range decoded.Delay {
		if delay != outfitGIFTwoFrameDelay {
			t.Fatalf("frame %d: expected delay %d, got %d", i, outfitGIFTwoFrameDelay, delay)
		}
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

func TestConvertOutfitSpritePNGToGIFPreservesTransparentPixels(t *testing.T) {
	sprite := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	sprite.Set(1, 0, color.NRGBA{R: 255, A: 255})
	sprite.Set(3, 1, color.NRGBA{B: 255, A: 255})

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
	if got := decoded.Image[0].ColorIndexAt(0, 0); got != 0 {
		t.Fatalf("expected transparent index 0 on first frame background, got %d", got)
	}
	if got := decoded.Image[1].ColorIndexAt(0, 0); got != 0 {
		t.Fatalf("expected transparent index 0 on second frame background, got %d", got)
	}

	transparent := decoded.Image[0].Palette[0]
	_, _, _, alpha := transparent.RGBA()
	if alpha != 0 {
		t.Fatalf("expected palette index 0 to be transparent, alpha=%d", alpha)
	}
}

func TestConvertOutfitSpritePNGToGIFKeepsDefaultDelayForMultiFrameSprites(t *testing.T) {
	sprite := image.NewNRGBA(image.Rect(0, 0, 16, 2))
	for frame := 0; frame < 8; frame++ {
		sprite.Set((frame*2)+1, 1, color.NRGBA{R: uint8(frame * 10), A: 255})
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
	if len(decoded.Image) != 8 {
		t.Fatalf("expected 8 gif frames, got %d", len(decoded.Image))
	}
	for i, delay := range decoded.Delay {
		if delay != outfitGIFDefaultFrameDelay {
			t.Fatalf("frame %d: expected delay %d, got %d", i, outfitGIFDefaultFrameDelay, delay)
		}
	}
}
