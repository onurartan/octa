package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/qeesung/image2ascii/convert"
	"octa"
)

func printAsciiLogo() {

	img, _, err := image.Decode(bytes.NewReader(octa.LogoData))
	if err != nil {
		fmt.Println("OCTA SERVER")
		return
	}

	convertOptions := convert.DefaultOptions
	convertOptions.FixedWidth = 35
	convertOptions.FixedHeight = 17

	converter := convert.NewImageConverter()
	fmt.Print(converter.Image2ASCIIString(img, &convertOptions))
}
