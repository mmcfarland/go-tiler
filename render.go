package main

import (
	"code.google.com/p/draw2d/draw2d"
	"errors"
	"fmt"
	"github.com/paulsmith/gogeos/geos"
	"image"
)

func RenderTile(bbox Envelope, config *LayerConfig) (*image.RGBA, error) {
	i := image.NewRGBA(image.Rect(0, 0, w, h))
	gc := draw2d.NewGraphicContext(i)

	geoms, err := GetTileFeatures(config.Table, bbox, config.GetStrokeWidth())
	if err != nil {
		fmt.Println(err)
		return i, err
	}

	gc.SetLineWidth(config.GetStrokeWidth())
	gc.SetStrokeColor(config.GetStrokeColor())

	for _, wkb := range geoms {
		geom, err := geos.FromWKB(wkb)
		if err != nil {
			fmt.Println(err)
			return i, err
		}
		t, err := geom.Type()
		if err != nil {
			fmt.Println(err)
			return i, err
		}

		switch t {
		case geos.LINESTRING, geos.MULTILINESTRING:
			renderLine(gc, geom, bbox)
		case geos.POLYGON, geos.MULTIPOLYGON:
			renderPolygon(gc, geom, bbox)
		case geos.POINT, geos.MULTIPOINT:
			renderPoint(gc, geom, bbox, config)
		default:
			return nil, errors.New(fmt.Sprintf("Unknown Geom Type: %s", t))
		}

		if config.HasFillColor() {
			gc.SetFillColor(config.GetFillColor())
			gc.FillStroke()
		} else {
			gc.Stroke()
		}
	}

	return i, nil
}

func renderPoint(gc *draw2d.ImageGraphicContext, geom *geos.Geometry, bbox Envelope,
	config *LayerConfig) {
	coords, err := geom.Coords()
	if err != nil {
		fmt.Println(err)
	}

	for _, c := range coords {
		pt := GeoPToImgP(c, bbox)
		draw2d.Circle(gc, pt.X, pt.Y, config.GetRadius())
	}
}

func renderPolygon(gc *draw2d.ImageGraphicContext, geom *geos.Geometry, bbox Envelope) {
	// TODO: Does not handle holes
	shell, err := geom.Shell()
	if err != nil {
		renderLine(gc, geom, bbox)
		return
	}
	renderLine(gc, shell, bbox)
}

func renderLine(gc *draw2d.ImageGraphicContext, geom *geos.Geometry, bbox Envelope) {
	coords, _ := geom.Coords()
	for idx, c := range coords {
		pt := GeoPToImgP(c, bbox)
		if idx == 0 {
			gc.MoveTo(pt.X, pt.Y)
		} else {
			gc.LineTo(pt.X, pt.Y)
		}
	}
}
