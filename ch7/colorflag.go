package ch7

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

type Color struct {
	R, G, B uint8
}

func (c Color) String() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

type colorFlag struct {
	Color
}

func ColorFlag(name string, value Color, usage string) *Color {
	f := &colorFlag{value}
	flag.CommandLine.Var(f, name, usage)
	return &f.Color
}

func (f *colorFlag) Set(s string) error {
	// 检查输入的字符串是否符合颜色格式，例如 "#RRGGBB"
	if len(s) == 7 && s[0] == '#' {
		// 从 hex 字符串中解析 R、G、B 分量
		r, err1 := strconv.ParseUint(s[1:3], 16, 8)
		g, err2 := strconv.ParseUint(s[3:5], 16, 8)
		b, err3 := strconv.ParseUint(s[5:7], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return fmt.Errorf("invalid color value: %s", s)
		}

		f.Color = Color{uint8(r), uint8(g), uint8(b)}
		return nil
	}

	// 检查颜色名称
	switch strings.ToLower(s) {
	case "red":
		f.Color = Color{255, 0, 0}
	case "green":
		f.Color = Color{0, 255, 0}
	case "blue":
		f.Color = Color{0, 0, 255}
	case "black":
		f.Color = Color{0, 0, 0}
	case "white":
		f.Color = Color{255, 255, 255}
	default:
		return fmt.Errorf("invalid color name: %s", s)
	}

	return nil

}

func (f *colorFlag) String() string {
	return f.Color.String()
}

var color = ColorFlag("color", Color{0, 0, 0}, "specify a color")

func TestColorFlag() {
	flag.Parse()
	fmt.Printf("Selected color: %s\n", color)
}
