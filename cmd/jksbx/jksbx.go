package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/png"
	"jksbx/pkg/captcha"
	"jksbx/pkg/cas"
	"net/http"
	"os"
)

//go:embed model.bin
var defaultModelData []byte

func main() {
	fmt.Println("Hello from jksbx.")

	m, err := captcha.LoadModel(bytes.NewReader(defaultModelData))
	if err != nil {
		fmt.Println(err)
	}
	captcha.Initialize(m)

	var tgc *http.Cookie = nil
	for turn := 0; turn < 5; turn++ {
		var captchaImage image.Image
		var jsessionid *http.Cookie
		var err error
		capt := ""

		for tries := 0; tries < 10; tries++ {
			captchaImage, jsessionid, err = cas.NewSessionAndGetRawCaptcha()
			if err != nil {
				continue
			}

			capt = captcha.Recognize(captchaImage)
			if capt != "" {
				break
			}
		}

		tgc, err = cas.LoginCas("NetID", "Password", capt, jsessionid)
		if err != nil {
			continue
		}

		if tgc != nil {
			break
		}
	}

	if tgc == nil {
		fmt.Println("TGC not get")
		return
	}
}

// InteractiveTrain将会开始进行交互式的训练模式，不断的下载新的验证码图片到
// 给定的filename中，并提示用户从stdin输入验证码。结束后，返回训练的模型。
func InteractiveTrain(filename string) captcha.Model {
	fmt.Println("开始训练模型，退出请输入q，撤销前一轮训练请输入x")
	m := captcha.Model{}
	var bufferImage image.Image = nil
	var bufferLabel string = ""
	numTrained := 0
	numFailed := 0
	reader := bufio.NewReader(os.Stdin)

	for {
		captchaImage, _, err := cas.NewSessionAndGetRawCaptcha()
		if err != nil {
			fmt.Println("获取验证码失败：" + err.Error())
			continue
		}

		file, err := os.Create(filename)
		if err != nil {
			fmt.Println("无法获取文件：" + err.Error())
			continue
		}
		if err = png.Encode(file, captchaImage); err != nil {
			fmt.Println("无法保存验证码图片：" + err.Error())
		}

		fmt.Printf("请输入%s所表示的验证码: ", filename)
		text, _ := reader.ReadString('\n')
		text = text[0 : len(text)-1]

		switch text {
		case "q":
			if bufferImage != nil {
				m.AddTrainingData(bufferImage, bufferLabel)
			}
			return m
		case "x":
			bufferImage = nil
			bufferLabel = ""
		default:
			if bufferImage != nil {
				numTrained++
				if !m.AddTrainingData(bufferImage, bufferLabel) {
					numFailed++
					fmt.Printf("上一张图片训练失败，目前失败率：%d/%d (%f%%)\n", numFailed, numTrained, 100*float64(numFailed)/float64(numTrained))
				}
			}
			for len(text) != 4 {
				fmt.Printf("请重新输入%s所表示的验证码: ", filename)
				text, _ = reader.ReadString('\n')
				text = text[0 : len(text)-1]
			}
			bufferImage = captchaImage
			bufferLabel = text
		}
	}
}
