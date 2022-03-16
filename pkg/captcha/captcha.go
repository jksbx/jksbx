/*
captcha包实现了cas系统验证码的OCR。
*/
package captcha

import (
	"image"
)

var (
	deltaX                     = [8]int{0, 1, 1, 1, 0, -1, -1, -1}
	deltaY                     = [8]int{1, 1, 0, -1, -1, -1, 0, 1}
	meanModel map[rune]feature = map[rune]feature{}
)

type feature struct {
	Width   float64
	Height  float64
	N       float64
	Numbers [4]float64
}

// Initialize用给定的模型数据来初始化模型。
func Initialize(m Model) {
	for char, modelElement := range m {
		s := float64(modelElement.Samples)
		feat := feature{}
		sum := modelElement.SumFeature
		feat.Width = sum.Width / s
		feat.Height = sum.Height / s
		feat.N = sum.N / s
		for i, num := range sum.Numbers {
			feat.Numbers[i] = num / s
		}

		meanModel[char] = feat
	}
}

// Recognize将识别给定的图片，返回4位小写字母和数字的组合。失败返回空串。
func Recognize(captcha image.Image) string {
	// 把有效字符抠出来。
	chars := denoiseAndSplit(captcha)
	if chars == nil {
		return ""
	}
	res := make([]rune, 0, 4)
	for _, char := range chars {
		// 连通块过小则为噪声。
		if len(char) <= 15 {
			continue
		}
		// 已经有4位字符了，再来就是有问题。
		if len(res) == 4 {
			return ""
		}

		feat := extractFeatures(char)
		if feat == nil {
			return ""
		}
		ch := recognizeSingle(feat)
		if ch == 0 {
			return ""
		}
		res = append(res, ch)
	}

	if len(res) != 4 {
		return ""
	}
	return string(res)
}

// extractFeatures对一个单独的字符提取特征，若失败返回nil。
func extractFeatures(char []image.Point) *feature {
	feat := feature{}
	feat.N = float64(len(char))

	left, right, up, down := 0xFFFF, -1, 0xFFFF, -1
	for _, p := range char {
		if left > p.X {
			left = p.X
		}
		if right < p.X {
			right = p.X
		}
		if up > p.Y {
			up = p.Y
		}
		if down < p.Y {
			down = p.Y
		}
	}
	feat.Width = float64(right - left + 1)
	feat.Height = float64(down - up + 1)

	cx := (left + right) / 2
	cy := (up + down) / 2
	for _, p := range char {
		idx := 0
		if p.X > cx {
			idx |= 1
		}
		if p.Y > cy {
			idx |= 2
		}
		feat.Numbers[idx] += 1.0
	}

	return &feat
}

// recognizeSingle对给定的特征进行识别，返回识别结果，若失败则返回0。
func recognizeSingle(feat *feature) rune {
	// 分数越低约好
	score := 1e50
	ret := rune(0)
	for char, modelFeat := range meanModel {
		dis := 0.0
		for i, num := range modelFeat.Numbers {
			dis += sqr(num - feat.Numbers[i])
		}

		if score > dis {
			score = dis
			ret = char
		}
	}

	return ret
}

// sqr计算平方。
func sqr(x float64) float64 {
	return x * x
}
