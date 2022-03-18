package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"jksbx/cmd/jksbx/router"
	"jksbx/internal/pkg/jlog"
	"jksbx/pkg/captcha"
	"net/http"
)

//go:embed model.bin
var defaultModelData []byte

func main() {
	fmt.Println("Hello from jksbx.")

	// 加载OCR模型数据并初始化模型。
	m, err := captcha.LoadModel(bytes.NewReader(defaultModelData))
	if err != nil {
		panic(err)
	}
	captcha.Initialize(m)

	// 初始化WEB服务器。
	router.InitializeApiEndpoints(false)
	address := ":8080"
	jlog.Infof("服务器启动，地址为：%s", address)
	err = http.ListenAndServe(address, nil)
	if err != nil {
		panic(err)
	}
}
