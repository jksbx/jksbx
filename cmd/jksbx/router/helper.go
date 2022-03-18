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
	tgc, jsessionid := loginCas(username, password)

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

// checkPasswordFromCas试图用指定帐号密码登录cas系统，以此来检查密码是否正确。注意如果返回false，
// 仍然有小概率密码不是错误的，可以检查密码确认无误后重试一次。
func checkPasswordFromCas(username, password string) bool {
	jlog.Infof("%s开始通过cas系统检查密码是否正确", username)
	tgc, _ := loginCas(username, password)
	return tgc != nil
}

// loginCas试图登录cas系统，返回TGC和JSESSIONID。若失败，TGC为nil。
func loginCas(username, password string) (*http.Cookie, *http.Cookie) {
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

	return tgc, jsessionid
}
