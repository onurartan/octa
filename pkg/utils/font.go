package utils

import (
	"fmt"
		"octa/pkg/logger"

	"os"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var (
	parsedFont *opentype.Font
	initMu     sync.Mutex
)


func InitFonts(fontPath string) error {
	initMu.Lock()
	defer initMu.Unlock()
	if parsedFont != nil {
		return nil
	}
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("failed to read font file: %w", err)
	}
	parsedFont, err = opentype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("failed to parse font file: %w", err)
	}
	return nil
}

func GetFont(fontPath string, size int) font.Face {
	if parsedFont == nil {
		logger.LogWarn("⚠️ Font not initialized! Call InitFonts first.")
		return nil
	}

	face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		logger.LogError("failed to create font face for size %d: %v", size, err)
		return nil
	}

	return face
}
