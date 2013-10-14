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
	StrokeWidth float64
	StrokeColor string
	FillColor   []string
}

func (c *LayerConfig) GetStrokeColor() color.NRGBA {
	rgba := parseColorString(c.StrokeColor)
	fmt.Println(rgba)
	return rgba
}

func parseColorString(s string) color.NRGBA {
	rgba := []uint8{0, 0, 0, 1}
	for i, v := range strings.Split(s, "'") {
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
