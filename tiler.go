package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"

	"code.google.com/p/draw2d/draw2d"
	"image"
	"image/png"
)

type Point struct {
	X, Y float64
}

type Envelope struct {
	Min, Max Point
}

const (
	w       = 256.0
	h       = 256.0
	mapSize = 20037508.34789244 * 2.0
)

var (
	Origin = Point{-20037508.34789244, 20037508.34789244}
)

func TileToBbox(xc, yc, zoom int) (bbox Envelope) {
	x := float64(xc)
	y := float64(yc)
	z := float64(zoom)
	size := mapSize / math.Pow(2, float64(z))
	fmt.Println(x, y, z, size)
	fmt.Println(x * size)
	bbox.Min = Point{Origin.X + x*size, Origin.Y - (y+1)*size}
	bbox.Max = Point{Origin.X + (x+1)*size, Origin.Y - y*size}
	return
}

func GeoPToImgP(geoP Point, b Envelope) Point {
	left := b.Min.X
	top := b.Max.Y
	x := (geoP.X - left) / w
	y := (top - geoP.Y) / h

	return Point{x, y}
}

func saveToPngFile(filePath string, m image.Image) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	b := bufio.NewWriter(f)
	err = png.Encode(b, m)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	err = b.Flush()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s OK.\n", filePath)
}

func main() {
	i := image.NewRGBA(image.Rect(0, 0, w, h))
	gc := draw2d.NewGraphicContext(i)
	b := Envelope{Point{2650000, 200000}, Point{2750000, 300000}}
	p := Point{2691389, 253794}

	sp1 := GeoPToImgP(p, b)
	sp2 := GeoPToImgP(Point{2699389, 253994}, b)

	gc.MoveTo(sp1.X, sp1.Y)
	gc.LineTo(sp2.X, sp2.Y)
	gc.Stroke()

	saveToPngFile("TestPath.png", i)

	m := TileToBbox(76, 97, 8)
	fmt.Printf("min: %f, max: %f", m.Min, m.Max)
}
