package utils

import (
	// "octa/pkg/logger"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	_ = godotenv.Load()
	// if err != nil {
	// 	logger.LogError("Failed to load .env file, check environment variables manually.")
	// }
}
