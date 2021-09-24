package main

import (
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/TarasLykhenko/tron/tron"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

var VERSION = "0.0.0-src"

func run() {
	cfg := pixelgl.WindowConfig{
		Title:  "Pixel Rocks!",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	win.Clear(colornames.Skyblue)

	p := tron.NewPlayer(1, "Taras")

	c := tron.Config{
		Width:      60,
		Height:     60,
		MaxPlayers: 6,
		// Mode:         "kd",
		// KickDeaths:   5,
		GameSpeed:    40 * time.Millisecond,
		RespawnDelay: 2 * time.Second,
		DBLocation:   filepath.Join(os.TempDir(), "tron.db"),
		GameWindow:   win,
	}

	rand.Seed(time.Now().UnixNano())

	g, err := tron.NewGame(c)
	g.AddPlayer(p)
	if err != nil {
		log.Fatal(err)
	}
	g.Play()

	for !win.Closed() {
		win.Update()
	}
}

func main() {

	pixelgl.Run(run)
}
