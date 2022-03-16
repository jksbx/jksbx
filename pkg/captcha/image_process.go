package captcha

import (
	"image"
	"image/color"
	"image/png"
	"io"
)

// DebugDenoise将提取给定的验证码图片中字符的部分，将提取出的像素描为纯红色，
// 数据写入outData中，格式为PNG。若错误则返回错误。
func DebugDenoise(captcha image.Image, outData io.Writer) error {
	bounds := captcha.Bounds()
	nw := image.NewRGBA(bounds)
	for i := 0; i < bounds.Dx(); i++ {
		for j := 0; j < bounds.Dy(); j++ {
			c := captcha.At(i, j)
			if checkValid(c) {
				nw.SetRGBA(i, j, color.RGBA{255, 0, 0, 255})
			} else {
				r, g, b, _ := c.RGBA()
				nw.SetRGBA(i, j, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255})
			}
		}
	}

	nwImage := nw.SubImage(bounds)
	if err := png.Encode(outData, nwImage); err != nil {
		return err
	}
	return nil
}

// denoiseAndSplit对验证码图像进行去噪处理，并分割成若干个字符，返回表示这几个
// 字符的切片，每个切片元素是一系列坐标点。
func denoiseAndSplit(captcha image.Image) [][]image.Point {
	ret := make([][]image.Point, 0, 4)

	bounds := captcha.Bounds()
	visited := map[int]struct{}{}
	for i := 0; i < bounds.Dx(); i++ {
		for j := 0; j < bounds.Dy(); j++ {
			points := bfs(captcha, i, j, visited)
			if points != nil {
				ret = append(ret, points)
			}
		}
	}

	return ret
}

// checkValid检查给定的颜色是否属于验证码中的有效字符。
func checkValid(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	e1, e2, e3 := uint(r), uint(g), uint(b)
	sqrsum := e1*e1 + e2*e2 + e3*e3
	return sqrsum > 80000000 && sqrsum < 5000000000
}

// bfs对给定的单元格作为起点开始BFS，返回这个连通块的所有坐标。如果起点不属于有效字符，
// 则返回nil。
func bfs(captcha image.Image, initialX, initialY int, visited map[int]struct{}) []image.Point {
	f := func(x, y int) int {
		return x*captcha.Bounds().Dy() + y
	}

	if _, ok := visited[f(initialX, initialY)]; ok {
		return nil
	}
	visited[f(initialX, initialY)] = struct{}{}
	if !checkValid(captcha.At(initialX, initialY)) {
		return nil
	}

	ret := make([]image.Point, 0, 20)
	que := make(chan image.Point, 100)
	que <- image.Point{initialX, initialY}
	for {
		select {
		case h := <-que:
			ret = append(ret, h)
			for k := 0; k < 8; k++ {
				tx := h.X + deltaX[k]
				ty := h.Y + deltaY[k]
				if tx < 0 || tx > captcha.Bounds().Dx() || ty < 0 || ty > captcha.Bounds().Dy() {
					continue
				}
				if _, ok := visited[f(tx, ty)]; ok {
					continue
				}
				if !checkValid(captcha.At(tx, ty)) {
					continue
				}
				visited[f(tx, ty)] = struct{}{}
				select {
				case que <- image.Point{tx, ty}:
				// 这种情况应该是出问题了，理论上缓冲区应该足够大，不可能爆。
				default:
					return nil
				}
			}

		default:
			return ret
		}
	}
}
