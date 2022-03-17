package jksb

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	JKSB_LOGIN_URL = "https://cas.sysu.edu.cn/cas/login?service=http://jksb.sysu.edu.cn/infoplus/login?retUrl=http://jksb.sysu.edu.cn/infoplus/form/XNYQSB/start"
)

type Session struct {
	timeoutCtx    context.Context
	timeoutCancel context.CancelFunc
	allocCtx      context.Context
	clientCtx     context.Context

	numRemainedPosts int
	postSet          map[network.RequestID]struct{}
	waitingDone      chan struct{}
}

// NewSession新建一个新的会话。
func NewSession(timeout time.Duration) *Session {
	ret := &Session{}
	ret.timeoutCtx, ret.timeoutCancel = context.WithTimeout(context.Background(), timeout)

	ret.allocCtx, _ = chromedp.NewExecAllocator(
		ret.timeoutCtx,

		append(
			chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false),
			chromedp.Flag("enable-automation", false),
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
		)...,
	)

	ret.clientCtx, _ = chromedp.NewContext(ret.allocCtx)

	ret.postSet = map[network.RequestID]struct{}{}
	ret.waitingDone = make(chan struct{})

	// 注册网络监听函数。
	chromedp.ListenTarget(ret.clientCtx, func(v interface{}) {
		switch ev := v.(type) {
		case *network.EventRequestWillBeSent:
			if ev.Request.Method == "POST" {
				ret.postSet[ev.RequestID] = struct{}{}
				fmt.Println("WillBeSent", ev.Request.URL, ev.RequestID, len(ret.postSet), ret.numRemainedPosts)
			}
		case *network.EventLoadingFailed:
			if _, ok := ret.postSet[ev.RequestID]; ok {
				ret.numRemainedPosts--
				if ret.numRemainedPosts == 0 {
					ret.waitingDone <- struct{}{}
				}
				fmt.Println("LoadingFailed", ev.RequestID, len(ret.postSet), ret.numRemainedPosts)
			}
		case *network.EventLoadingFinished:
			if _, ok := ret.postSet[ev.RequestID]; ok {
				ret.numRemainedPosts--
				if ret.numRemainedPosts == 0 {
					ret.waitingDone <- struct{}{}
				}
				fmt.Println("LoadingFinished", ev.RequestID, len(ret.postSet), ret.numRemainedPosts)
			}
		}
	})

	return ret
}

// LoginJksb将登录进jksb系统。
// func (s *Session) LoginJksb(username, password string) error {
// 	// 获取验证码数据。
// 	var captchaBytes []byte
// 	err := chromedp.Run(s.clientCtx,
// 		chromedp.Navigate(JKSB_LOGIN_URL),
// 		chromedp.Screenshot("#captchaImg", &captchaBytes, chromedp.NodeVisible),
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	captchaImage, err := png.Decode(bytes.NewReader(captchaBytes))
// 	if err != nil {
// 		return err
// 	}

// 	capt := captcha.Recognize(captchaImage)
// 	err = chromedp.Run(s.clientCtx,
// 		chromedp.WaitVisible("#username", chromedp.ByID),
// 		chromedp.SetValue("#username", username, chromedp.ByID),
// 		chromedp.WaitVisible("#password", chromedp.ByID),
// 		chromedp.SetValue("#password", password, chromedp.ByID),
// 		chromedp.WaitVisible("#captcha", chromedp.ByID),
// 		chromedp.SetValue("#captcha", capt, chromedp.ByID),
// 		chromedp.Click(`//*[@value="LOGIN"]`, chromedp.BySearch),
// 		chromedp.Click(`#form_command_bar > li:first-child > a`, chromedp.ByQuery),
// 	)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	return nil
// }

// LoginJksb用TGC和JSESSIONID登入jksb系统，注意fakeHeader应该与登录cas时的一致。
func (s *Session) LoginJksb(tgc, jsessionid *http.Cookie, fakeHeader map[string]string) error {
	temFakeHeader := map[string]interface{}{}
	for k, v := range fakeHeader {
		temFakeHeader[k] = v
	}

	// 页面加载好之后再模拟点击。
	s.numRemainedPosts = 3
	err := chromedp.Run(s.clientCtx,
		network.SetExtraHTTPHeaders(network.Headers(temFakeHeader)),
		setCookie(tgc.Name, tgc.Value, "cas.sysu.edu.cn", "/cas/", true, true),
		setCookie(jsessionid.Name, jsessionid.Value, "cas.sysu.edu.cn", "/cas", true, false),
		chromedp.Navigate(JKSB_LOGIN_URL),
	)
	if err != nil {
		return err
	}

	<-s.waitingDone

	return nil
}

func setCookie(name, value, domain, path string, httpOnly, secure bool) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		return network.SetCookie(name, value).
			WithDomain(domain).
			WithPath(path).
			WithHTTPOnly(httpOnly).
			WithSecure(secure).
			Do(ctx)
	})
}

// SubmitJksb将试图模拟提交申报表操作，注意此时会话必须处于申报表填写页面，正常情况下，LoginJksb成功后，
// 页面就处于申报表填写页面。
func (s *Session) SubmitJksb() error {
	// 后面模拟点击“下一步”后，开始监听POST请求的数量，完成了一定次数后代表第二步的页面
	// 已经加载好，可以进行后续操作。
	s.numRemainedPosts = 4

	// 提交申报表的第一步（阅读相关信息）。
	err := chromedp.Run(s.clientCtx,
		chromedp.Click(`#form_command_bar > li:first-child > a`, chromedp.ByQuery),
	)
	if err != nil {
		return err
	}
	<-s.waitingDone

	// 后面模拟“提交”后，再次阻塞，等待指定次数POST请求结束后，代表表单提交完成，可以关闭浏览器了。
	s.numRemainedPosts = 2

	// 已经加载好第二步，这里模拟点击“提交”。
	err = chromedp.Run(s.clientCtx,
		chromedp.Click(`#form_command_bar > li:first-child > a`, chromedp.ByQuery),
	)
	if err != nil {
		return err
	}
	<-s.waitingDone

	s.timeoutCancel()

	return nil
}
