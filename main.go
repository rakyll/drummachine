// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event"
	"golang.org/x/mobile/f32"
	"golang.org/x/mobile/geom"
	"golang.org/x/mobile/gl"
	"golang.org/x/mobile/gl/glutil"
)

const (
	numBeats  = 12
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
	touchLoc geom.Point
)

func main() {
	app.Run(app.Callbacks{
		Start: start,
		Stop:  stop,
		Draw:  draw,
		Touch: touch,
	})
}

func start() {
	var err error
	program, err = glutil.CreateProgram(vertexShader, fragmentShader)
	if err != nil {
		log.Printf("error creating GL program: %v", err)
		return
	}

	buf = gl.GenBuffer()
	gl.BindBuffer(gl.ARRAY_BUFFER, buf)
	gl.BufferData(gl.ARRAY_BUFFER, gl.STATIC_DRAW, rectData)

	position = gl.GetAttribLocation(program, "position")
	color = gl.GetUniformLocation(program, "color")
	offset = gl.GetUniformLocation(program, "offset")
	// touchLoc = geom.Point{geom.Width / 2, geom.Height / 2}
	//

	go func() {
		for {
			index = (index + 1) % numBeats
			time.Sleep(300 * time.Millisecond)
		}
	}()

	hitData[0][3] = true
	hitData[0][6] = true
	hitData[3][3] = true

	hitData[0][4] = true
	hitData[2][4] = true
	hitData[4][4] = true
	hitData[6][4] = true
	hitData[8][4] = true
	hitData[10][4] = true
}

func stop() {
	gl.DeleteProgram(program)
	gl.DeleteBuffer(buf)
}

func touch(t event.Touch) {
	touchLoc = t.Loc
	fmt.Println(t)
}

var rectData = f32.Bytes(binary.LittleEndian,
	0, 0,
	0, 0.1,
	0.1, 0,
	0.1, 0.1,
)

var hitData [numBeats][numTracks]bool

func draw() {
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.UseProgram(program)

	if greenDec {
		green -= 0.01
		if green <= 0.2 {
			greenDec = false
		}
	} else {
		green += 0.01
		if green >= 0.5 {
			greenDec = true
		}
	}

	for i := 0; i < numBeats; i++ {
		for j := 0; j < numTracks; j++ {
			var c float32
			switch {
			case hitData[i][j]:
				c = 1
			case i == index:
				c = green
			default:
				c = 0
			}
			drawButton(c, float32(i)*1/numBeats, float32(j)*1/numTracks)
		}
	}

	//debug.DrawFPS()
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
