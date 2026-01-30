package utils

import (
	"mime/multipart"
	"net/http"
)

func IsImageFile(fileHeader *multipart.FileHeader) bool {
	file, err := fileHeader.Open()
	if err != nil {
		return false
	}
	defer file.Close()

	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		return false
	}

	contentType := http.DetectContentType(buff)

	allowed := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		// "image/webp": true,
	}

	return allowed[contentType]
}
