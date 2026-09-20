package main

import (
	"github.com/bidirekt/cli/internal"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	internal.Run()
}
