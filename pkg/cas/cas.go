/*
cas包实现了鸭大cas系统的模拟登录。
*/
package cas

import (
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	CAS_LOGIN_URL = "https://cas.sysu.edu.cn/cas/login"
	CAS_CAPTCHA   = "https://cas.sysu.edu.cn/cas/captcha.jsp"
)

// LoginCas 用给定的用户名，密码，验证码来登录cas系统，注意登录前需要先
// 获取一次验证码。返回登录态cookie，若登录失败则为nil。
func LoginCas(username, password, captcha string, jsessionid *http.Cookie, fakeHeader map[string]string) (*http.Cookie, error) {
	form := url.Values{
		"username":    []string{username},
		"password":    []string{password},
		"captcha":     []string{captcha},
		"_eventId":    []string{"submit"},
		"geolocation": []string{""},
		"execution":   []string{""},
	}

	// 找到HTML源码里的execution，登录表单需要用到。
	req, err := http.NewRequest("GET", CAS_LOGIN_URL, nil)
	if err != nil {
		return nil, err
	}
	addHeaders(req, fakeHeader)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body := string(bodyBytes)
	start := strings.Index(body, "execution")
	if start == -1 {
		return nil, fmt.Errorf("登录页面HTML源码不包含execution")
	}
	start += 18
	end := start + 1
	for end < len(body) && body[end] != '"' {
		end++
	}
	if end == len(body) {
		return nil, fmt.Errorf("登录页面HTML的execution不包含右引号")
	}
	form["execution"][0] = body[start:end]

	// 构造请求的Header和Cookie
	req, err = http.NewRequest("POST", CAS_LOGIN_URL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	addHeaders(req, fakeHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(jsessionid)

	// 正式发起登录请求。
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// 检查是否登录成功，若成功，则响应Cookie里有TGC
	var tgc *http.Cookie = nil
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "TGC" {
			tgc = cookie
			break
		}
	}

	return tgc, nil
}

// NewSessionAndGetRawCaptcha新起一个会话，获得验证码，返回这个验证码图片，以及此次会话的JSESSIONID。
func NewSessionAndGetRawCaptcha(fakeHeader map[string]string) (image.Image, *http.Cookie, error) {
	req, err := http.NewRequest("GET", CAS_CAPTCHA, nil)
	if err != nil {
		return nil, nil, err
	}
	addHeaders(req, fakeHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	ret, err := jpeg.Decode(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var jsessionid *http.Cookie = nil
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "JSESSIONID" {
			jsessionid = cookie
			break
		}
	}
	if jsessionid == nil {
		return nil, nil, fmt.Errorf("获取captcha时未得到JSESSIONID")
	}

	return ret, jsessionid, nil
}

// addHeaders伪造头部，用于通过安全检查，获取可用TGC。细节请看文档。
func addHeaders(req *http.Request, fakeHeader map[string]string) {
	for k, v := range fakeHeader {
		req.Header.Add(k, v)
	}
}
