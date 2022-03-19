package main

import (
	"bytes"
	_ "embed"
	"flag"
	"jksbx/cmd/jksbx/router"
	"jksbx/internal/pkg/jlog"
	"jksbx/internal/pkg/userdb"
	"jksbx/pkg/captcha"
	"jksbx/pkg/everyday"
	"net/http"
	"time"
)

//go:embed model.bin
var defaultModelData []byte

func main() {
	// 解析命令行参数。
	headfulMode := flag.Bool("e", false, "是否需要有头浏览器，忽略则为不需要，即用无头浏览器提交健康申报表")
	everydayHm := flag.Int("s", 730, "每天开始自动申报的时间，格式为24小时制HHMM，如七点半为730，晚上八点整为2000")
	address := flag.String("a", ":8080", "WEB服务的监听地址，默认监听 0.0.0.0:8080")
	userDataFilename := flag.String("u", "user.db", "用户数据库文件路径，忽略则为当前目录的user.db")
	modelFilename := flag.String("m", "", "OCR模型文件路径，忽略则使用内嵌默认模型")
	flag.Parse()

	// 加载OCR模型数据并初始化模型。
	var m captcha.Model
	var err error
	if *modelFilename == "" {
		m, err = captcha.LoadModel(bytes.NewReader(defaultModelData))
	} else {
		m, err = captcha.LoadModelFile(*modelFilename)
	}
	if err != nil {
		panic(err)
	}
	captcha.Initialize(m)

	// 初始化userdb并启动服务。
	userdb.Initialize(*userDataFilename)
	userdb.StartAutoJob(time.Hour)

	// 初始化每日健康申报任务。
	hour := *everydayHm / 100
	minute := *everydayHm % 100
	if hour < 0 || hour >= 24 || minute < 0 || minute >= 60 {
		panic("开始申报时间格式不正确")
	}
	everyday.StartEverydayJob(hour, minute, router.EveryoneSubmitJksb)

	// 初始化WEB服务器。
	router.InitializeApiEndpoints(*headfulMode)
	jlog.Infof("服务器启动，地址为：%s", *address)
	err = http.ListenAndServe(*address, nil)
	if err != nil {
		panic(err)
	}
}
