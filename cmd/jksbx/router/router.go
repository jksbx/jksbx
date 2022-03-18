package router

import (
	"jksbx/internal/pkg/userdb"
	"net/http"
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

	// POST /api/test 接收username和password，并进行一次提交健康申报表的测试。
	http.HandleFunc("/api/test", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			rw.WriteHeader(405)
			rw.Write([]byte("请求非POST方法"))
			return
		}
		username, password, err := getUserInfoFromForm(r)
		if err != nil {
			rw.WriteHeader(400)
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

	// POST /api/adduser 接收username和password，存入后台数据库中。如果已经存在了，则为no-op。
	http.HandleFunc("/api/adduser", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			rw.WriteHeader(405)
			rw.Write([]byte("请求非POST方法"))
			return
		}
		username, password, err := getUserInfoFromForm(r)
		if err != nil {
			rw.WriteHeader(400)
			rw.Write([]byte(err.Error()))
			return
		}

		if userdb.ExistsUser(username) {
			rw.WriteHeader(406)
			rw.Write([]byte("账户已经存在，不可添加。修改密码请先用旧密码删除，再添加"))
			return
		}

		if !checkPasswordFromCas(username, password) {
			rw.WriteHeader(406)
			rw.Write([]byte("密码可能不正确，请检查密码后重试"))
			return
		}

		userdb.AddUser(username, password)
		rw.Write([]byte("done"))
	})

	// POST /api/deleteuser 接收username和password，如果密码正确，则删除用户。
	http.HandleFunc("/api/deleteuser", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			rw.WriteHeader(405)
			rw.Write([]byte("请求非POST方法"))
			return
		}
		username, password, err := getUserInfoFromForm(r)
		if err != nil {
			rw.WriteHeader(400)
			rw.Write([]byte(err.Error()))
			return
		}

		if !userdb.CheckUser(username, password) {
			rw.WriteHeader(406)
			rw.Write([]byte("密码错误，或账户已经不在数据库中"))
			return
		}

		userdb.DeleteUser(username)
		rw.Write([]byte("done"))
	})
}
