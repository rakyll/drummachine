// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// A simple drum machine app.
//
// Note: This demo is an early preview of Go 1.5. In order to build this
// program as an Android APK using the gomobile tool, you need to install
// Go 1.5 from the source.
//
// Clone the source from the tip under $HOME/go directory. On Windows,
// you may like to clone the repo to your user folder, %USERPROFILE%\go.
//
//   $ git clone https://go.googlesource.com/go $HOME/go
//
// Go 1.5 requires Go 1.4. Read more about this requirement at
// http://golang.org/s/go15bootstrap.
// Set GOROOT_BOOTSTRAP to the GOROOT of your existing 1.4 installation or
// follow the steps below to checkout go1.4 from the source and build.
//
//   $ git clone https://go.googlesource.com/go $HOME/go1.4
//   $ cd $HOME/go1.4
//   $ git checkout go1.4.1
//   $ cd src && ./make.bash
//
// If you clone Go 1.4 to a different destination, set GOROOT_BOOTSTRAP
// environmental variable accordingly.
//
// Build Go 1.5 and add Go 1.5 bin to your path.
//
//   $ cd $HOME/go/src && ./make.bash
//   $ export PATH=$PATH:$HOME/go/bin
//
// Set a GOPATH if no GOPATH is set, add $GOPATH/bin to your path.
//
//   $ export GOPATH=$HOME
//   $ export PATH=$PATH:$GOPATH/bin
//
// Get the gomobile tool and initialize.
//
//   $ go get golang.org/x/mobile/cmd/gomobile
//   $ gomobile init
//
// It may take a while to initialize gomobile, please wait.
//
// Get the drum machine example and use gomobile to build or install it on your device.
//
//   $ go get -d github.com/rakyll/drummachine
//   $ gomobile build github.com/rakyll/drummachine # will build an APK
//
//   # plug your Android device to your computer or start an Android emulator.
//   # if you have adb installed on your machine, use gomobile install to
//   # build and deploy the APK to an Android target.
//   $ gomobile install github.com/rakyll/drummachine
//
// Switch to your device or emulator to start the Drum Machine application from
// the launcher.
// You can also run the application on your desktop by running the command
// below. (Note: It currently doesn't work on Windows.)
//   $ go install github.com/rakyll/drummachine && drummachine
package main

import (
	"fmt"
	"image"
	"io"
	"log"
	"time"

	_ "image/jpeg"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/audio"
	"golang.org/x/mobile/event"
	"golang.org/x/mobile/f32"
	"golang.org/x/mobile/gl"
	"golang.org/x/mobile/sprite"
	"golang.org/x/mobile/sprite/clock"
	"golang.org/x/mobile/sprite/glsprite"
)

var (
	startClock = time.Now()
	lastClock  = clock.Time(-1)
	eng        = glsprite.Engine()

	texs []sprite.SubTex

	board *sprite.Node
)

var (
	buttons [4][4]bool
	pattern [8][16]bool
	index   int

	samples [16]io.Closer
	players [16]*audio.Player
)

func main() {
	// TODO(jbd): Handle touch to turn on/off the beats.
	app.Run(app.Callbacks{
		Start: start,
		Stop:  stop,
		Draw:  draw,
		Touch: touch,
	})
}

// TODO(jbd): Add multitouch.

func touch(t event.Touch) {
	x, y := float32(t.Loc.X), float32(t.Loc.Y)
	i := int((x - offset) / button)
	j := int((y - offset) / button)
	if i < 0 || i > 3 || j < 0 || j > 3 {
		return
	}
	if t.Type == event.TouchStart {
		glow(i, j)
		pattern[index][i*4+j] = true
	}
}

func start() {
	for i := 0; i < len(samples); i++ {
		src, err := app.Open(fmt.Sprintf("sample%d.wav", i))
		if err != nil {
			log.Fatal(err)
		}
		samples[i] = src
		p, err := audio.NewPlayer(src, 0, 0)
		if err != nil {
			log.Fatal(err)
		}
		players[i] = p
	}

	// player goroutine
	go func() {
		for {
			for i := 0; i < 4; i++ {
				for j := 0; j < 4; j++ {
					if pattern[index][i*4+j] {
						glow(i, j)
					}
				}
			}
			// bpm=140
			time.Sleep(time.Minute / 140)
			index = (index + 1) % len(pattern)
		}
	}()
}

// TODO(jbd): racy glow, do we have to care?

func glow(i, j int) {
	p := players[j*4+i]
	buttons[i][j] = true
	p.Seek(0)
	p.Play()
	go func() {
		time.Sleep(300 * time.Millisecond)
		buttons[i][j] = false
	}()
}

func stop() {
	for _, p := range players {
		p.Close()
	}
	for _, s := range samples {
		s.Close()
	}
}

// TODO(jbd): Should work in potrait mode.

var (
	offset float32
	button float32
)

func draw() {
	if texs == nil {
		texs = loadTextures()
	}

	h := app.GetConfig().Height
	button = float32(h / 5)
	offset = button / 2

	now := clock.Time(time.Since(startClock) * 60 / time.Second)
	if now == lastClock {
		// TODO: figure out how to limit draw callbacks to 60Hz instead of
		// burning the CPU as fast as possible.
		// TODO: (relatedly??) sync to vblank?
		return
	}
	lastClock = now
	gl.ClearColor(1, 1, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	cfg := app.GetConfig()
	board = &sprite.Node{}
	eng.Register(board)
	eng.SetTransform(board, f32.Affine{
		{1, 0, 0},
		{0, 1, 0},
	})

	n := newNode()
	eng.SetSubTex(n, texs[texBG])
	eng.SetTransform(n, f32.Affine{
		{float32(cfg.Width), 0, 0},
		{0, float32(cfg.Height), 0},
	})
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			drawButton(i, j)
		}
	}

	drawBrandModel()
	eng.Render(board, now)
}

func drawButton(i, j int) {
	n := newNode()
	if buttons[i][j] {
		eng.SetSubTex(n, texs[texButtonOn])
	} else {
		eng.SetSubTex(n, texs[texButtonOff])
	}
	margin := button / 10
	eng.SetTransform(n, f32.Affine{
		{button - margin, 0, offset + float32(i)*button},
		{0, button - margin, offset + float32(j)*button},
	})
}

func drawBrandModel() {
	n := newNode()
	eng.SetSubTex(n, texs[texBrand])
	eng.SetTransform(n, f32.Affine{
		{56, 0, 5 * button},
		{0, 22, offset},
	})
}

func newNode() *sprite.Node {
	n := &sprite.Node{}
	eng.Register(n)
	board.AppendChild(n)
	return n
}

const (
	texBG = iota
	texButtonOn
	texButtonOff
	texBrand
	texModel
	texOthers
)

func loadTextures() []sprite.SubTex {
	a, err := app.Open("sprite.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	img, _, err := image.Decode(a)
	if err != nil {
		log.Fatal(err)
	}
	t, err := eng.LoadTexture(img)
	if err != nil {
		log.Fatal(err)
	}

	return []sprite.SubTex{
		texBG:        sprite.SubTex{t, image.Rect(0, 0, 24, 860)},
		texButtonOff: sprite.SubTex{t, image.Rect(94, 242, 94+150, 242+151)},
		texButtonOn:  sprite.SubTex{t, image.Rect(94, 413, 94+150, 413+151)},
		texBrand:     sprite.SubTex{t, image.Rect(94, 31, 94+227, 31+88)},
		texModel:     sprite.SubTex{t, image.Rect(162, 120, 162+140, 120+90)},
		texOthers:    sprite.SubTex{t, image.Rect(162, 120, 162+140, 120+90)},
	}
}
