// 生成 GitPilot Logo — 极简风格：圆角深色背景 + 大号 Git 分支符号
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const size = 1024

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	bg := color.RGBA{30, 30, 46, 255}       // #1e1e2e
	blue := color.RGBA{137, 180, 250, 255}  // #89b4fa
	green := color.RGBA{166, 227, 161, 255} // #a6e3a1

	cx := size / 2
	cy := size / 2

	// 1. 圆角背景
	roundedRect(img, 0, 0, size, size, 220, bg)

	// 2. 主干竖线 — 粗实线
	thickLine(img, cx, cy-280, cx, cy+280, 50, blue)

	// 3. 分支曲线 — 从主干中上方向右上弯出
	smoothBranch(img, cx, cy-60, cx+200, cy-260, 40, green)

	// 4. 三个实心节点
	filledCircle(img, cx, cy-260, 56, blue)      // 顶部
	filledCircle(img, cx, cy+260, 56, blue)      // 底部
	filledCircle(img, cx+200, cy-260, 48, green) // 分支端

	// 保存
	save(img, "build/appicon.png")
	save(img, "frontend/public/appicon.png")
	println("✅ GitPilot Logo 已生成!")
}

// --- 绘图工具 ---

func roundedRect(img *image.RGBA, x0, y0, x1, y1, r int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dx, dy := 0, 0
			if x < x0+r && y < y0+r {
				dx, dy = x0+r-x, y0+r-y
			} else if x > x1-r-1 && y < y0+r {
				dx, dy = x-(x1-r-1), y0+r-y
			} else if x < x0+r && y > y1-r-1 {
				dx, dy = x0+r-x, y-(y1-r-1)
			} else if x > x1-r-1 && y > y1-r-1 {
				dx, dy = x-(x1-r-1), y-(y1-r-1)
			}
			dist := math.Sqrt(float64(dx*dx + dy*dy))
			if dist <= float64(r)+0.5 {
				if dist > float64(r)-1 {
					blend(img, x, y, c, float64(r)+0.5-dist)
				} else {
					img.SetRGBA(x, y, c)
				}
			}
		}
	}
}

func filledCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r - 1; y <= cy+r+1; y++ {
		for x := cx - r - 1; x <= cx+r+1; x++ {
			dist := math.Hypot(float64(x-cx), float64(y-cy))
			if dist <= float64(r)+0.5 {
				if dist > float64(r)-1 {
					blend(img, x, y, c, float64(r)+0.5-dist)
				} else {
					img.SetRGBA(x, y, c)
				}
			}
		}
	}
}

func thickLine(img *image.RGBA, x0, y0, x1, y1, thickness int, c color.RGBA) {
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	steps := int(length * 1.5)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		px := float64(x0) + dx*t
		py := float64(y0) + dy*t
		filledCircle(img, int(math.Round(px)), int(math.Round(py)), thickness/2, c)
	}
}

func smoothBranch(img *image.RGBA, x0, y0, x1, y1, thickness int, c color.RGBA) {
	// 二次贝塞尔: 控制点使起点水平、终点竖直
	cpx := float64(x1)
	cpy := float64(y0)

	steps := 120
	prevX, prevY := float64(x0), float64(y0)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		u := 1.0 - t
		px := u*u*float64(x0) + 2*u*t*cpx + t*t*float64(x1)
		py := u*u*float64(y0) + 2*u*t*cpy + t*t*float64(y1)
		thickLine(img, int(prevX), int(prevY), int(px), int(py), thickness, c)
		prevX, prevY = px, py
	}
}

func blend(img *image.RGBA, x, y int, c color.RGBA, alpha float64) {
	if x < 0 || y < 0 || x >= size || y >= size || alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	e := img.RGBAAt(x, y)
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(float64(e.R)*(1-alpha) + float64(c.R)*alpha),
		G: uint8(float64(e.G)*(1-alpha) + float64(c.G)*alpha),
		B: uint8(float64(e.B)*(1-alpha) + float64(c.B)*alpha),
		A: uint8(math.Min(255, float64(e.A)+float64(c.A)*alpha)),
	})
}

func save(img *image.RGBA, path string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, img)
	println("  保存:", path)
}
