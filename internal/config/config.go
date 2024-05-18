package config

import (
	"os"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load("env.ini")
	if err != nil {
		panic("Error loading .env file")
	}
}

func Get(key string) string {
	return os.Getenv(key)
}
