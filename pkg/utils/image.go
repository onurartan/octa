package utils

import (
	"crypto/md5"
	"octa/pkg/logger"

	"fmt"

	"image"
	"image/color"

	"strings"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// GenerateGradient: Operates in 3 different modes.
// 1. pallete="pro"   -> Selects from the list
// 2. pallete="retro" -> MD5 Raw
// 3. pallete="vivid" -> HSL Math
// palette: “pro” | ‘retro’ | “auto”
func GenerateGradient(name string, palette string) (color.RGBA, color.RGBA) {
	switch strings.ToLower(palette) {
	case "pro", "curated":
		return generateGradientFromList(name) // ProGradients listesinden
	case "retro", "raw":
		return generateGradientRetro(name) // MD5 Ham
	case "vivid", "auto", "":
		return generateGradientProcedural(name) // HSL Matematik
	default:
		return generateGradientProcedural(name)
	}
}

func generateGradientRetro(name string) (color.RGBA, color.RGBA) {
	hash := md5.Sum([]byte(name))

	c1 := color.RGBA{R: hash[0], G: hash[1], B: hash[2], A: 255}
	c2 := color.RGBA{R: hash[3], G: hash[4], B: hash[5], A: 255}

	return c1, c2
}

func generateGradientProcedural(name string) (color.RGBA, color.RGBA) {

	hash := md5.Sum([]byte(name))

	h1 := float64(int(hash[0])) * (360.0 / 255.0)
	s1 := 0.65 + (float64(hash[1]%35) / 100.0)
	l1 := 0.45 + (float64(hash[2]%20) / 100.0)

	hueShift := 30.0 + float64(hash[3]%60)
	h2 := h1 + hueShift
	if h2 > 360 {
		h2 -= 360
	}

	s2 := 0.65 + (float64(hash[4]%35) / 100.0)
	l2 := 0.45 + (float64(hash[5]%20) / 100.0)

	r1, g1, b1 := hslToRgb(h1, s1, l1)
	r2, g2, b2 := hslToRgb(h2, s2, l2)

	return color.RGBA{r1, g1, b1, 255}, color.RGBA{r2, g2, b2, 255}
}

// Gradient Selection by Name
// generate gradient from ProGradients list
func generateGradientFromList(name string) (color.RGBA, color.RGBA) {
	hash := 0
	for _, c := range name {
		hash = int(c) + ((hash << 5) - hash)
	}
	if hash < 0 {
		hash = -hash
	}
	pair := ProGradients[hash%len(ProGradients)]
	return pair.Start, pair.End
}

// SOLID COLOR SELECTOR
// palette: "pro" | "google" | "auto"
func GetColorFromPalette(name string, palette string) color.RGBA {
	var targetList []color.RGBA

	switch strings.ToLower(palette) {
	case "google", "brand":
		targetList = GoogleColors
	case "pro", "curated":
		targetList = ProColors
	default:
		targetList = ProColors
	}

	hash := 0
	for _, c := range name {
		hash = int(c) + ((hash << 5) - hash)
	}
	if hash < 0 {
		hash = -hash
	}

	return targetList[hash%len(targetList)]
}



func GetInitials(name string) string {
	var initials string

	words := strings.Fields(name)
	for _, word := range words {

		if len(word) > 0 {
			runes := []rune(word)
			initials += string(unicode.ToUpper(runes[0]))
		}

		if len([]rune(initials)) >= 2 {
			break
		}
	}

	// Eğer hiç harf çıkmadıysa (örn: sembol girildiyse) ismin ilk harfini zorla al
	if len(initials) == 0 && len(name) > 0 {
		runes := []rune(name)
		initials = string(unicode.ToUpper(runes[0]))
	}

	return initials
}

func GetTextColor(bg color.RGBA) string {
	// >_ constrart for text color
	luminance := 0.299*float64(bg.R) + 0.587*float64(bg.G) + 0.114*float64(bg.B)
	if luminance > 186 {
		return "black"
	}
	return "white"
}

func DominantFromGradient(c1, c2 color.RGBA) color.RGBA {
	return color.RGBA{
		R: uint8((int(c1.R) + int(c2.R)) / 2),
		G: uint8((int(c1.G) + int(c2.G)) / 2),
		B: uint8((int(c1.B) + int(c2.B)) / 2),
		A: 255,
	}
}

func Luminance(c color.RGBA) float64 {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	return 0.2126*r + 0.7152*g + 0.0722*b
}

func DetermineTextColorAdvanced(
	c1 color.RGBA,
	c2 color.RGBA,
	aType string,
	input string,
) color.Color {

	
	if input != "" {
		
		switch strings.ToLower(input) {
		case "white":
			return color.White
		case "black":
			return color.Black
		}

		
		if c, err := ParseColor(input); err == nil {
			return c
		}
	}


	var base color.RGBA

	if aType == "gradient" {
		base = DominantFromGradient(c1, c2)
	} else {
		base = c1
	}

	if Luminance(base) > 0.6 {
		return color.Black
	}

	return color.White
}

func DetermineTextColor(bg color.RGBA, input string) color.Color {
	switch strings.ToLower(input) {
	case "white":
		return color.White
	case "black":
		return color.Black
	default:
		if GetTextColor(bg) == "black" {
			return color.Black
		}
		return color.White
	}
}

func CalculateFontSize(size int, text string) int {
	base := float64(size) * 0.6 

	switch len([]rune(text)) {
	case 1:
		return int(base)
	case 2:
		return int(base * 0.72)
	default:
		return int(base * 0.63)
	}
}

func GenerateSVG(
	size int,
	name string,
	bg1, bg2 color.RGBA,
	text string,
	rounded int,
	textColor color.Color,
	aType string, // "gradient", "soft", "color"
) string {

	if aType == "" {
		aType = "color"
	}

	if text == "auto" {
		text = GetInitials(name)
	}

	fill := "white"
	if textColor != nil {
		r, g, b, _ := textColor.RGBA()
		fill = fmt.Sprintf("rgb(%d,%d,%d)", r>>8, g>>8, b>>8)
	}

	fontSize := CalculateFontSize(size, text)

	textSVG := ""
	if text != "" {
		textSVG = fmt.Sprintf(`
	<text
		x="50%%"
		y="50%%"
		text-anchor="middle"
		dominant-baseline="central"
		font-family="Inter, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
		font-weight="600"
		font-size="%d"
		fill="%s"
		letter-spacing="-0.03em"
	>%s</text>`, fontSize, fill, text)
	}

	if aType == "soft" || aType == "color" {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">
	<rect width="%d" height="%d" rx="%d" ry="%d" fill="rgb(%d,%d,%d)" />
	%s
</svg>`,
			size, size, size, size,
			size, size, rounded, rounded,
			bg1.R, bg1.G, bg1.B,
			textSVG,
		)
	}

	// Gradient
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">
	<defs>
		<linearGradient id="gradient" x1="1" y1="1" x2="0" y2="0">
			<stop offset="0%%" stop-color="rgb(%d,%d,%d)" />
			<stop offset="100%%" stop-color="rgb(%d,%d,%d)" />
		</linearGradient>
	</defs>
	<rect width="%d" height="%d" rx="%d" ry="%d" fill="url(#gradient)" />
	%s
</svg>`,
		size, size, size, size,
		bg1.R, bg1.G, bg1.B,
		bg2.R, bg2.G, bg2.B,
		size, size, rounded, rounded,
		textSVG,
	)
}

func DrawText(img *image.RGBA, text string, textColor color.Color, size int) {
	col := textColor

	fontSize := int(float64(size) / 2)
	loadedFont := GetFont("fonts/Inter_24pt-Medium.ttf", fontSize)
	if loadedFont == nil {
		logger.LogError("Font failed to load. Unable to draw text.")
		return
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: loadedFont,
	}

	textWidth := d.MeasureString(text).Round()

	metrics := loadedFont.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	textHeight := ascent + descent

	// >_ Postion(Center)
	x := (size - textWidth) / 2
	y := (size-textHeight)/2 + ascent

	d.Dot = fixed.P(x, y)
	d.DrawString(text)
}
