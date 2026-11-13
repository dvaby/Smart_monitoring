package main

import (
	"embed"
	"log"

	"snm/cmd/snm"
)

//go:embed all:frontend/dist build/appicon.png
var assets embed.FS

func main() {
	if err := snm.Run(assets); err != nil {
		log.Fatal(err)
	}
}
