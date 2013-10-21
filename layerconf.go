package main

import (
	"code.google.com/p/gcfg"
	"fmt"
	"image/color"
	"log"
	"strconv"
	"strings"
)

type LayerConfig struct {
	Table       string
	StrokeWidth float64
	StrokeColor string
	FillColor   string
	Radius      float64
	fcol        color.NRGBA
	fcolParsed  bool
	scol        color.NRGBA
	scolParsed  bool
}

func (c *LayerConfig) GetStrokeColor() color.NRGBA {
	if c.scolParsed == false {
		c.scol = parseColorString(c.StrokeColor)
		c.scolParsed = true
	}
	return c.scol
}

func (c *LayerConfig) GetFillColor() color.NRGBA {
	if c.fcolParsed == false {
		c.fcol = parseColorString(c.FillColor)
		c.fcolParsed = true
	}
	return c.fcol
}

func (c *LayerConfig) HasFillColor() bool {
	return c.FillColor != ""
}

func (c *LayerConfig) GetRadius() float64 {
	if c.Radius == 0.0 {
		return 1.0
	}
	return c.Radius
}

func parseColorString(s string) color.NRGBA {
	rgba := []uint8{0, 0, 0, 0}
	for i, v := range strings.Split(s, ",") {
		x, err := strconv.Atoi(v)
		if err == nil && x < 256 {
			rgba[i] = uint8(x)
		} else {
			log.Fatal("Bad color config")
		}
	}

	return color.NRGBA{rgba[0], rgba[1], rgba[2], rgba[3]}
}

func (c *LayerConfig) GetStrokeWidth() float64 {
	if c.StrokeWidth > 0 {
		return c.StrokeWidth
	}
	return 1
}

type Config struct {
	Database struct {
		Name string
		User string
		Host string
	}
	Layer map[string]*LayerConfig
}

func ParseConfig() (*Config, error) {
	var Conf Config
	err := gcfg.ReadFileInto(&Conf, "settings.conf")
	if err != nil {
		fmt.Println("Invalid setting.conf file", err)
		return &Conf, err
	}

	return &Conf, nil
}
