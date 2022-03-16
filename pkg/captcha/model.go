package captcha

import (
	"encoding/gob"
	"image"
	"io"
	"os"
)

type modelElement struct {
	Samples    int
	SumFeature feature
}

type Model map[rune]*modelElement

// AddTrainingData新增一张验证码图片的数据，返回是否成功。
func (m Model) AddTrainingData(captchaImage image.Image, labels string) bool {
	if len(labels) != 4 {
		return false
	}

	chars := denoiseAndSplit(captchaImage)
	if chars == nil {
		return false
	}
	feats := make([]*feature, 0, 4)
	for _, char := range chars {
		// 连通块过小则为噪声。
		if len(char) <= 15 {
			continue
		}
		// 已经有4位字符了，再来就是有问题。
		if len(feats) == 4 {
			return false
		}

		feat := extractFeatures(char)
		if feat == nil {
			return false
		}

		feats = append(feats, feat)
	}

	if len(feats) != 4 {
		return false
	}

	for i := 0; i < 4; i++ {
		feat := feats[i]
		label := rune(labels[i])
		if _, ok := m[label]; !ok {
			m[label] = &modelElement{}
		}

		m[label].Samples++
		sum := &m[label].SumFeature
		sum.Width += feat.Width
		sum.Height += feat.Height
		sum.N += feat.N
		for j := range sum.Numbers {
			sum.Numbers[j] += feat.Numbers[j]
		}
	}
	return true
}

// DumpModel将一个内存中的模型写入给定Writer中，出错则返回错误。
func DumpModel(m Model, w io.Writer) error {
	enc := gob.NewEncoder(w)
	return enc.Encode(m)
}

// DumpModel将一个内存中的模型写入给定文件中，出错则返回错误。
func DumpModelFile(m Model, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	return DumpModel(m, f)
}

// LoadModel将从给定Reader中加载模型，出错则返回错误。
func LoadModel(r io.Reader) (Model, error) {
	dec := gob.NewDecoder(r)
	m := Model{}
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

// LoadModelFile将从给定文件中加载模型，出错则返回错误。
func LoadModelFile(filename string) (Model, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return LoadModel(f)
}
