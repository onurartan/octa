package utils

import (
	"bytes"
	"github.com/disintegration/imaging"
	"image"
	"image/jpeg"
)

type ProcessOptions struct {
	Mode    string // "square", "fit", "original", "scale"
	Size    int    // Pixel-based size (256, 512, etc.)
	Scale   int    // Percentage-based size (1-100)
	Quality int
}

func ProcessImage(img image.Image, opts ProcessOptions) (*bytes.Buffer, int, int, error) {
	var finalImg image.Image

	switch opts.Mode {
	case "square":
		// Make a square and cut it in half
		finalImg = imaging.Fill(img, opts.Size, opts.Size, imaging.Center, imaging.Lanczos)

	case "fit":
		// Fit to pixel limit (e.g., maximum 1024px)
		if img.Bounds().Dx() > opts.Size || img.Bounds().Dy() > opts.Size {
			finalImg = imaging.Fit(img, opts.Size, opts.Size, imaging.Lanczos)
		} else {
			finalImg = img
		}

	case "scale":
		if opts.Scale <= 0 || opts.Scale >= 100 {
			finalImg = img
		} else {
			// Math: New Width = Old Width * (Percentage / 100)
			width := img.Bounds().Dx() * opts.Scale / 100
			height := img.Bounds().Dy() * opts.Scale / 100

			if width < 1 {
				width = 1
			}
			if height < 1 {
				height = 1
			}

			finalImg = imaging.Resize(img, width, height, imaging.Lanczos)
		}

	case "original":
		finalImg = img

	default:
		finalImg = imaging.Fill(img, 256, 256, imaging.Center, imaging.Lanczos)
	}

	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, finalImg, &jpeg.Options{Quality: opts.Quality})

	return buf, finalImg.Bounds().Dx(), finalImg.Bounds().Dy(), err
}
