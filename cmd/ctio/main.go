package main

import (
	"github.com/contracttesting/cli/internal"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	internal.Run()
}
