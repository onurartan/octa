package utils

import (
	"errors"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// --- YAPILAR ---

type SoftColorPair struct {
	Background color.RGBA
	Text       color.RGBA
}

type GradientPair struct {
	Start color.RGBA
	End   color.RGBA
}

var cssColors = map[string]color.RGBA{
	"white":   {255, 255, 255, 255},
	"black":   {0, 0, 0, 255},
	"red":     {255, 0, 0, 255},
	"green":   {0, 128, 0, 255},
	"blue":    {0, 0, 255, 255},
	"cyan":    {0, 255, 255, 255},
	"magenta": {255, 0, 255, 255},
	"yellow":  {255, 255, 0, 255},
	"orange":  {255, 165, 0, 255},
	"purple":  {128, 0, 128, 255},
	"pink":    {255, 192, 203, 255},
	"gray":    {128, 128, 128, 255},
	"grey":    {128, 128, 128, 255},
	"silver":  {192, 192, 192, 255},
	"gold":    {255, 215, 0, 255},
	"teal":    {0, 128, 128, 255},
	"lime":    {0, 255, 0, 255},
	"navy":    {0, 0, 128, 255},
}

// ProColors: ~100 Selected Modern Colors
var ProColors = []color.RGBA{
	{71, 85, 105, 255}, {51, 65, 85, 255}, {30, 41, 59, 255}, // Slate
	{82, 82, 91, 255}, {63, 63, 70, 255}, {39, 39, 42, 255}, // Zinc
	{87, 83, 78, 255}, {68, 64, 60, 255}, {41, 37, 36, 255}, // Stone
	{75, 85, 99, 255}, {55, 65, 81, 255}, {31, 41, 55, 255}, // Gray

	{239, 68, 68, 255}, {220, 38, 38, 255}, {185, 28, 28, 255}, // Red
	{244, 63, 94, 255}, {225, 29, 72, 255}, {190, 18, 60, 255}, // Rose
	{236, 72, 153, 255}, {219, 39, 119, 255}, {190, 24, 93, 255}, // Pink

	{249, 115, 22, 255}, {234, 88, 12, 255}, {194, 65, 12, 255}, // Orange
	{245, 158, 11, 255}, {217, 119, 6, 255}, {180, 83, 9, 255}, // Amber
	{234, 179, 8, 255}, {202, 138, 4, 255}, {161, 98, 7, 255}, // Yellow (Koyu tonlar)

	{34, 197, 94, 255}, {22, 163, 74, 255}, {21, 128, 61, 255}, // Green
	{16, 185, 129, 255}, {5, 150, 105, 255}, {4, 120, 87, 255}, // Emerald
	{132, 204, 22, 255}, {101, 163, 13, 255}, {77, 124, 15, 255}, // Lime

	{20, 184, 166, 255}, {13, 148, 136, 255}, {15, 118, 110, 255}, // Teal
	{6, 182, 212, 255}, {8, 145, 178, 255}, {21, 94, 117, 255}, // Cyan
	{14, 165, 233, 255}, {2, 132, 199, 255}, {3, 105, 161, 255}, // Sky

	{59, 130, 246, 255}, {37, 99, 235, 255}, {29, 78, 216, 255}, // Blue
	{99, 102, 241, 255}, {79, 70, 229, 255}, {67, 56, 202, 255}, // Indigo
	{139, 92, 246, 255}, {124, 58, 237, 255}, {109, 40, 217, 255}, // Violet

	// --- PURPLE & FUCHSIA (Yaratıcılık) ---
	{168, 85, 247, 255}, {147, 51, 234, 255}, {126, 34, 206, 255}, // Purple
	{217, 70, 239, 255}, {192, 38, 211, 255}, {162, 28, 175, 255}, // Fuchsia

	{88, 101, 242, 255}, // Blurple (Discord)
	{29, 161, 242, 255}, // Twitter Blue
	{0, 0, 0, 255},      // Pure Black
	{25, 25, 25, 255},   // Off Black
}

// ProGradients: Selected Gradients
var ProGradients = []GradientPair{
	{Start: color.RGBA{59, 130, 246, 255}, End: color.RGBA{37, 99, 235, 255}},  // Blue -> Dark Blue
	{Start: color.RGBA{139, 92, 246, 255}, End: color.RGBA{124, 58, 237, 255}}, // Violet -> Deep Violet
	{Start: color.RGBA{236, 72, 153, 255}, End: color.RGBA{219, 39, 119, 255}}, // Pink -> Rose
	{Start: color.RGBA{16, 185, 129, 255}, End: color.RGBA{5, 150, 105, 255}},  // Emerald -> Green
	{Start: color.RGBA{249, 115, 22, 255}, End: color.RGBA{234, 88, 12, 255}},  // Orange -> Red Orange
	{Start: color.RGBA{99, 102, 241, 255}, End: color.RGBA{168, 85, 247, 255}}, // Indigo -> Purple
	{Start: color.RGBA{6, 182, 212, 255}, End: color.RGBA{59, 130, 246, 255}},  // Cyan -> Blue
	{Start: color.RGBA{244, 63, 94, 255}, End: color.RGBA{249, 115, 22, 255}},  // Rose -> Orange
	{Start: color.RGBA{34, 197, 94, 255}, End: color.RGBA{20, 184, 166, 255}},  // Green -> Teal
	{Start: color.RGBA{71, 85, 105, 255}, End: color.RGBA{30, 41, 59, 255}},    // Slate -> Dark Slate
	{Start: color.RGBA{168, 85, 247, 255}, End: color.RGBA{236, 72, 153, 255}}, // Purple -> Pink
	{Start: color.RGBA{14, 165, 233, 255}, End: color.RGBA{99, 102, 241, 255}}, // Sky -> Indigo
}

var GoogleColors = []color.RGBA{

	{R: 59, G: 130, B: 246, A: 255}, // Royal Blue
	{R: 37, G: 99, B: 235, A: 255},  // Darker Blue
	{R: 14, G: 165, B: 233, A: 255}, // Sky Blue
	{R: 6, G: 182, B: 212, A: 255},  // Cyan

	{R: 139, G: 92, B: 246, A: 255}, // Violet
	{R: 124, G: 58, B: 237, A: 255}, // Deep Violet
	{R: 192, G: 38, B: 211, A: 255}, // Fuchsia
	{R: 219, G: 39, B: 119, A: 255}, // Pink
	{R: 225, G: 29, B: 72, A: 255},  // Rose

	{R: 16, G: 185, B: 129, A: 255}, // Emerald
	{R: 5, G: 150, B: 105, A: 255},  // Forest Green
	{R: 20, G: 184, B: 166, A: 255}, // Teal
	{R: 13, G: 148, B: 136, A: 255}, // Dark Teal

	{R: 249, G: 115, B: 22, A: 255}, // Orange
	{R: 234, G: 88, B: 12, A: 255},  // Burnt Orange
	{R: 245, G: 158, B: 11, A: 255}, // Amber
	{R: 220, G: 38, B: 38, A: 255},  // Red

	{R: 71, G: 85, B: 105, A: 255}, // Slate
	{R: 82, G: 82, B: 91, A: 255},  // Zinc
	{R: 79, G: 70, B: 229, A: 255}, // Indigo
}

// GetSoftColorPair: Selects a color from ProColors by name
// and automatically converts it to the Soft format (Light Background, Dark Text).
func GetSoftColorPair(name string, pallete string) SoftColorPair {
	baseColor := GetColorFromPalette(name, pallete)

	return MakeSoft(baseColor)
}

// MakeSoft: Takes any color, preserves the Hue value
// Lightens the background, darkens the text.
func MakeSoft(seed color.RGBA) SoftColorPair {
	h, s, _ := rgbToHsl(seed.R, seed.G, seed.B)

	bgR, bgG, bgB := hslToRgb(h, math.Min(s, 0.6), 0.95)

	textR, textG, textB := hslToRgb(h, math.Min(s+0.2, 1.0), 0.20)

	return SoftColorPair{
		Background: color.RGBA{bgR, bgG, bgB, 255},
		Text:       color.RGBA{textR, textG, textB, 255},
	}
}

func SoftDarken(c color.RGBA, factor float64) color.RGBA {

	h, s, l := rgbToHsl(c.R, c.G, c.B)
	l = math.Max(0, l-factor)

	r, g, b := hslToRgb(h, s, l)
	return color.RGBA{r, g, b, 255}
}

func ParseColor(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return color.RGBA{}, errors.New("empty color string")
	}

	lowerName := strings.ToLower(s)
	if c, ok := cssColors[lowerName]; ok {
		return c, nil
	}

	c := color.RGBA{A: 255}
	hexStr := strings.TrimPrefix(s, "#")

	switch len(hexStr) {
	case 6:
		r, err1 := strconv.ParseUint(hexStr[0:2], 16, 8)
		g, err2 := strconv.ParseUint(hexStr[2:4], 16, 8)
		b, err3 := strconv.ParseUint(hexStr[4:6], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return color.RGBA{}, errors.New("invalid hex")
		}
		c.R, c.G, c.B = uint8(r), uint8(g), uint8(b)
	case 3:
		r, err1 := strconv.ParseUint(string(hexStr[0])+string(hexStr[0]), 16, 8)
		g, err2 := strconv.ParseUint(string(hexStr[1])+string(hexStr[1]), 16, 8)
		b, err3 := strconv.ParseUint(string(hexStr[2])+string(hexStr[2]), 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return color.RGBA{}, errors.New("invalid hex")
		}
		c.R, c.G, c.B = uint8(r), uint8(g), uint8(b)
	default:
		return color.RGBA{}, errors.New("invalid color format")
	}

	return c, nil
}

func rgbToHsl(r, g, b uint8) (h, s, l float64) {
	rf, gf, bf := float64(r)/255.0, float64(g)/255.0, float64(b)/255.0
	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	l = (max + min) / 2.0

	if max == min {
		h, s = 0, 0
	} else {
		d := max - min
		s = d / (max + min)
		if l > 0.5 {
			s = d / (2.0 - max - min)
		}
		switch max {
		case rf:
			h = (gf - bf) / d
			if gf < bf {
				h += 6.0
			}
		case gf:
			h = (bf-rf)/d + 2.0
		case bf:
			h = (rf-gf)/d + 4.0
		}
		h *= 60.0
	}
	return
}

func hslToRgb(h, s, l float64) (r, g, b uint8) {
	var rf, gf, bf float64
	if s == 0 {
		rf, gf, bf = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q
		rf = hueToRgb(p, q, h/360.0+1.0/3.0)
		gf = hueToRgb(p, q, h/360.0)
		bf = hueToRgb(p, q, h/360.0-1.0/3.0)
	}
	return uint8(rf * 255), uint8(gf * 255), uint8(bf * 255)
}

func hueToRgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6.0*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6.0
	}
	return p
}
