package main 

import (
        "bufio"
        "fmt"
        "log"
        "os"

        "code.google.com/p/draw2d/draw2d"
        "image"
        "image/png"
)

const (
    w = 256
    h = 256
)

func GeoPToImgP (geoP image.Point, bounds image.Rectangle) image.Point {
    r := bounds.Canon()
    left := r.Min.X
    top := r.Max.Y
    x := (geoP.X - left) / w;
    y := (top - geoP.Y) / h;

    return image.Pt(x, y)
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
        i := image.NewRGBA(image.Rect(0, 0, 200, 200))
        gc := draw2d.NewGraphicContext(i)
        gc.MoveTo(10.0, 10.0)
        gc.LineTo(100.0, 10.0)
        gc.Stroke()
        saveToPngFile("TestPath.png", i)
}
