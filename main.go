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
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/audio"
	"golang.org/x/mobile/event"
	"golang.org/x/mobile/f32"
	"golang.org/x/mobile/geom"
	"golang.org/x/mobile/gl"
	"golang.org/x/mobile/gl/glutil"
)

const (
	numBeats  = 16
	numTracks = 8
)

var (
	program  gl.Program
	position gl.Attrib
	offset   gl.Uniform
	color    gl.Uniform
	buf      gl.Buffer

	index    int
	green    float32
	greenDec bool

	stopped bool
)

var (
	hits    [numBeats][numTracks]bool
	samples [numTracks]io.Closer
	players [numTracks]*audio.Player
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

func touch(t event.Touch) {
	// TODO(jbd): fix the vertex shader or the vertices,
	// the touches have to be slightly at the bottom region
	// of a particular button.
	if t.Type == event.TouchStart {
		x := int((t.Loc.X / geom.Width) * numBeats)
		y := int((t.Loc.Y / geom.Height) * numTracks)
		hits[x][y] = !hits[x][y]
	}
}

func start() {
	var err error
	program, err = glutil.CreateProgram(vertexShader, fragmentShader)
	if err != nil {
		log.Printf("error creating GL program: %v", err)
		return
	}

	buf = gl.CreateBuffer()
	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.BufferData(gl.ARRAY_BUFFER, rectData, gl.STATIC_DRAW)

	position = gl.GetAttribLocation(program, "position")
	color = gl.GetUniformLocation(program, "color")
	offset = gl.GetUniformLocation(program, "offset")

	for i := 0; i < numTracks; i++ {
		rc, err := app.Open(fmt.Sprintf("track%d.wav", i))
		if err != nil {
			log.Fatal(err)
		}
		samples[i] = rc
		p, err := audio.NewPlayer(rc, audio.Stereo16, 44100)
		if err != nil {
			log.Fatal(err)
		}
		players[i] = p
	}

	// hi hat
	hits[0][1] = true
	hits[2][1] = true
	hits[4][1] = true
	hits[6][1] = true
	hits[8][1] = true
	hits[10][1] = true
	hits[12][1] = true
	hits[14][1] = true

	// kick
	hits[5][2] = true
	hits[7][2] = true
	hits[11][2] = true
	hits[13][2] = true
	hits[14][2] = true
	hits[15][2] = true

	// bass
	hits[0][4] = true
	hits[3][4] = true
	hits[5][4] = true
	hits[6][4] = true
	hits[8][4] = true
	hits[11][4] = true
	hits[13][4] = true

	// bass2
	hits[2][6] = true
	hits[10][6] = true

	go func() {
		for {
			if stopped {
				stopped = false
				return
			}
			index = (index + 1) % numBeats
			for t := 0; t < numTracks; t++ {
				go func(t int) {
					if hits[index][t] {
						players[t].Play()
					}
				}(t)
			}
			// bpm=140
			time.Sleep(time.Minute / 140)
		}
	}()
}

func stop() {
	for _, p := range players {
		p.Close()
	}
	for _, s := range samples {
		s.Close()
	}
	gl.DeleteProgram(program)
	gl.DeleteBuffer(buf)
	stopped = true
}

var rectData = f32.Bytes(binary.LittleEndian,
	0, 0,
	0, 0.1,
	0.1, 0,
	0.1, 0.1,
)

func draw() {
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.UseProgram(program)

	if greenDec {
		green -= 0.01
	} else {
		green += 0.01
	}
	if green <= 0.2 {
		greenDec = false
	}
	if green >= 0.5 {
		greenDec = true
	}

	for i := 0; i < numBeats; i++ {
		for j := 0; j < numTracks; j++ {
			var c float32
			switch {
			case hits[i][j]:
				c = 1
			case i == index:
				c = green
			default:
				c = 0
			}
			drawButton(c, float32(i)*1/numBeats, float32(j)*1/numTracks)
		}
	}
}

func drawButton(g, x, y float32) {
	gl.Uniform4f(color, 0.1, g, 0.4, 1) // color
	gl.Uniform2f(offset, x, y)          // position

	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.EnableVertexAttribArray(position)
	gl.VertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	gl.DisableVertexAttribArray(position)
}

const vertexShader = `#version 100
uniform vec2 offset;
attribute vec4 position;
void main() {
  // offset comes in with x/y values between 0 and 1.
  // position bounds are -1 to 1.
  vec4 offset4 = vec4(2.0*offset.x-1.0, 1.0-2.0*offset.y, 0, 0);
  gl_Position = position + offset4;
}`

const fragmentShader = `#version 100
precision mediump float;
uniform vec4 color;
void main() {
  gl_FragColor = color;
}`
