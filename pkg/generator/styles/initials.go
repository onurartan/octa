package styles

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"octa/internal/config"
	"octa/pkg/utils"
)

const DefaultAvatarSize = 360

// ============================================================================
// 1. YENİ CORE FONKSİYON (MOTOR) ⚙️
// Sadece veri üretir, HTTP bilmez. Cache ve eski fonksiyon bunu çağırır.
// ============================================================================
func GenerateImageBytes(name string, query url.Values) ([]byte, string, error) {

	// Format
	format := "png"
	if f := query.Get("format"); f == "svg" || f == "png" {
		format = f
	} else if t := query.Get("type"); t == "svg" {
		format = "svg"
	}

	// Style
	style := "color"
	palette := "auto"
	if theme := query.Get("theme"); theme != "" {
		parts := strings.Split(theme, "/")
		if len(parts) > 0 {
			style = parts[0]
		}
		if len(parts) > 1 {
			palette = parts[1]
		}
	} else if at := query.Get("aType"); at != "" {
		style = at 
	}

	if style != "gradient" && style != "soft" {
		style = "color"
	}

	// name
	initials := query.Get("initials")
	if initials == "" || initials == "auto" {
		targetName := name
		if iName := query.Get("iName"); iName != "" {
			targetName = iName
		}
		initials = utils.GetInitials(targetName)
	}

	// size
	size := config.AppConfig.Image.DefaultSize
	if size == 0 {
		size = DefaultAvatarSize
	}
	if sVal := query.Get("size"); sVal == "" {
		sVal = query.Get("w")
	} else {
		if s, err := strconv.Atoi(sVal); err == nil {
			if s > 1024 {
				size = 1024
			} else if s < 16 {
				size = 16
			} else {
				size = s
			}
		}
	}

	// Calculate Color
	var bg1, bg2 color.RGBA
	var txtColor color.Color

	switch style {
	case "soft":
		seed := color.RGBA{0, 0, 0, 255}
		if palette == "auto" {
			seed, _ = utils.GenerateGradient(name, "auto")
		} else {
			seed = utils.GetColorFromPalette(name, palette)
		}
		pair := utils.MakeSoft(seed)
		bg1, txtColor = pair.Background, pair.Text
		bg2 = utils.SoftDarken(bg1, 0.05)
	case "gradient":
		bg1, bg2 = utils.GenerateGradient(name, palette)
		txtColor = utils.DetermineTextColorAdvanced(bg1, bg2, "gradient", "")
	default:
		c := utils.GetColorFromPalette(name, palette)
		bg1, bg2 = c, c
		txtColor = utils.DetermineTextColorAdvanced(bg1, bg2, "color", "")
	}

	// Override
	userHasBg := false
	if bgOv := query.Get("bg"); bgOv != "" {
		if c, err := utils.ParseColor(bgOv); err == nil {
			bg1, bg2 = c, c
			userHasBg = true
		}
	}
	if txtOv := query.Get("color"); txtOv != "" {
		txtColor = utils.DetermineTextColorAdvanced(bg1, bg2, style, txtOv)
	} else if userHasBg {
		txtColor = utils.DetermineTextColorAdvanced(bg1, bg2, "custom", "")
	}

	// Rounded
	var radius float64
	if rVal := query.Get("rounded"); rVal == "true" {
		radius = float64(size) / 16.0
	} else if rVal != "" {
		if v, err := strconv.Atoi(rVal); err == nil {
			if v > 50 {
				v = 50
			}
			radius = (float64(size) / 2.0) * (float64(v) / 100.0) * 2
		}
	}


	// SVG
	if format == "svg" {
		svgContent := utils.GenerateSVG(size, name, bg1, bg2, initials, int(radius), txtColor, style)
		return []byte(svgContent), "image/svg+xml", nil
	}

	// PNG (Pixel Perfect)
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	fSize := float64(size)
	rSq := radius * radius

	for y := 0; y < size; y++ {
		fy := float64(y) + 0.5
		for x := 0; x < size; x++ {
			if radius > 0 {
				fx := float64(x) + 0.5
				dx, dy := 0.0, 0.0
				isCorner := false
				if fx < radius && fy < radius {
					dx, dy, isCorner = fx-radius, fy-radius, true
				} else if fx > fSize-radius && fy < radius {
					dx, dy, isCorner = fx-(fSize-radius), fy-radius, true
				} else if fx < radius && fy > fSize-radius {
					dx, dy, isCorner = fx-radius, fy-(fSize-radius), true
				} else if fx > fSize-radius && fy > fSize-radius {
					dx, dy, isCorner = fx-(fSize-radius), fy-(fSize-radius), true
				}
				if isCorner && (dx*dx+dy*dy > rSq) {
					continue
				}
			}

			if bg1 == bg2 {
				img.SetRGBA(x, y, bg1)
			} else {
				ratio := (float64(x) + float64(y)) / (2 * fSize)
				r := uint8(float64(bg1.R)*(1-ratio) + float64(bg2.R)*ratio)
				g := uint8(float64(bg1.G)*(1-ratio) + float64(bg2.G)*ratio)
				b := uint8(float64(bg1.B)*(1-ratio) + float64(bg2.B)*ratio)
				img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
			}
		}
	}

	if initials != "" {
		utils.DrawText(img, initials, txtColor, size)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, "", fmt.Errorf("encode error: %v", err)
	}

	return buf.Bytes(), "image/png", nil
}

func GenerateInitialsAvatar(name string, w http.ResponseWriter, r *http.Request) {
	data, mimeType, err := GenerateImageBytes(name, r.URL.Query())

	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, utils.ErrServerInternal, err.Error())
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "public, max-age=604800") // 1 Hafta
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}
