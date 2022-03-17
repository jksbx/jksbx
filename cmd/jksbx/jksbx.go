package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/png"
	"jksbx/internal/pkg/jksb"
	"jksbx/pkg/captcha"
	"jksbx/pkg/cas"
	"os"
	"time"
)

//go:embed model.bin
var defaultModelData []byte

func main() {
	fmt.Println("Hello from jksbx.")

	fakeHeader := map[string]string{
		"Connection":                "keep-alive",
		"sec-ch-ua":                 `" Not A;Brand";v="99", "Chromium";v="99"`,
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-Linux":           `"Linux"`,
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.74 Safari/537.36",
		"Accept":                    `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`,
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-User":            "?1",
		"Sec-Fetch-Dest":            "document",
		"Accept-Encoding":           "gzip, deflate, br",
		"Accept-Language":           "en-US,en;q=0.9",
	}

	m, err := captcha.LoadModel(bytes.NewReader(defaultModelData))
	if err != nil {
		fmt.Println(err)
	}
	captcha.Initialize(m)

	captchaImage, jsessionid, _ := cas.NewSessionAndGetRawCaptcha(fakeHeader)
	capt := captcha.Recognize(captchaImage)
	if capt == "" {
		panic("captcha not recognized")
	}
	tgc, err := cas.LoginCas("NetID", "Password", capt, jsessionid, fakeHeader)
	if err != nil {
		panic(err)
	}
	if tgc == nil {
		panic("login fail")
	}

	s := jksb.NewSession(time.Minute * 2)
	err = s.LoginJksb(tgc, jsessionid, fakeHeader)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	err = s.SubmitJksb()
	if err != nil {
		fmt.Println("Error: " + err.Error())
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
		captchaImage, _, err := cas.NewSessionAndGetRawCaptcha(map[string]string{})
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
