package router

import (
	"fmt"
	"image"
	"jksbx/internal/pkg/jksb"
	"jksbx/internal/pkg/jlog"
	"jksbx/pkg/captcha"
	"jksbx/pkg/cas"
	"net/http"
	"time"
)

var fakeHeader map[string]string
var headful bool

// InitializeApiEndpoints将为所有API入口注册处理函数，需要指定后台提交申报表时，
// 是否需要显示浏览器界面（即是否要有头浏览器）。
func InitializeApiEndpoints(head bool) {
	fakeHeader = map[string]string{
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
	headful = head

	// /api/test 接收username和password，并进行一次提交健康申报表的测试。
	http.HandleFunc("/api/test", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			rw.WriteHeader(404)
			rw.Write([]byte("请求非POST方法"))
			return
		}
		username, password, err := getUserInfoFromForm(r)
		if err != nil {
			rw.WriteHeader(404)
			rw.Write([]byte(err.Error()))
			return
		}

		err = submitJskb(username, password)
		if err != nil {
			rw.WriteHeader(500)
			rw.Write([]byte(err.Error()))
			return
		}

		rw.Write([]byte("done"))
	})
}

// getUserInfoFromForm从请求体中获取用户账户名和密码。如果没找到相关信息，则返回错误。
func getUserInfoFromForm(r *http.Request) (string, string, error) {
	err := r.ParseForm()
	if err != nil {
		return "", "", err
	}

	username := r.PostFormValue("username")
	if username == "" {
		return "", "", fmt.Errorf("未填写用户名")
	}
	password := r.PostFormValue("password")
	if password == "" {
		return "", "", fmt.Errorf("未填写密码")
	}

	return username, password, nil
}

// submitJksb将根据账户名和密码尝试提交健康申报表。
func submitJskb(username, password string) error {
	jlog.Infof("%s开始提交申报表，开始登录cas系统", username)

	numTryLogin := 10
	numTryCaptcha := 30

	var tgc, jsessionid *http.Cookie
	for i := 0; i < numTryLogin; i++ {
		capt := ""
		for j := 0; j < numTryCaptcha; j++ {
			var captchaImage image.Image
			var err error

			captchaImage, jsessionid, err = cas.NewSessionAndGetRawCaptcha(fakeHeader)
			if err != nil {
				jlog.Warnf("%s获取验证码失败：%s", username, err.Error())
				break
			}
			capt = captcha.Recognize(captchaImage)
			if capt == "" {
				jlog.Warnf("%s验证码无法识别", username)
				continue
			}
		}

		if capt == "" {
			continue
		}

		var err error
		tgc, err = cas.LoginCas(username, password, capt, jsessionid, fakeHeader)
		if err != nil {
			jlog.Warnf("%s登录cas系统时出现问题", username)
			continue
		}
		if tgc == nil {
			jlog.Warnf("%s登录cas系统时，验证码识别失败，或密码错误", username)
			continue
		}
	}

	if tgc == nil {
		jlog.Errorf("%s登录cas系统失败", username)
		return fmt.Errorf("登录cas系统失败")
	}

	jlog.Infof("%s开始登录jksb系统并提交申报表", username)
	s := jksb.NewSession(time.Minute*2, fakeHeader["User-Agent"], headful)
	err := s.LoginJksb(tgc, jsessionid, fakeHeader)
	if err != nil {
		jlog.Errorf("%s登录jksb系统失败，有可能是网站下线了？%s", username, err.Error())
		return err
	}

	err = s.SubmitJksb()
	if err != nil {
		jlog.Errorf("%s提交申报表失败：%s", username, err.Error())
		return err
	}

	jlog.Infof("%s成功提交申报表", username)
	return nil
}
