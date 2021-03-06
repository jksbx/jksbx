package router

import (
	_ "embed"
	"fmt"
	"jksbx/internal/pkg/jlog"
	"jksbx/internal/pkg/userdb"
	"net/http"
	"sync"
	"time"
)

var fakeHeader map[string]string
var headful bool

//go:embed index.html
var indexPage []byte

type userInfo struct {
	username string
	password string
}

// InitializeApiEndpoints将为所有API入口注册处理函数，需要指定后台提交申报表时，
// 是否需要显示浏览器界面（即是否要有头浏览器）。
func InitializeApiEndpoints(head bool, queueSize, concurrency int) {
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

	requestQueue := make(chan userInfo, queueSize)
	inQueue := map[string]struct{}{}
	inQueueMutex := sync.RWMutex{}
	// 这个值不是那么重要，因此不加锁
	meanDuration := 10.0

	// 起若干个协程来并发处理请求。
	for i := 0; i < concurrency; i++ {
		go func(goroutineId int) {
			for {
				u := <-requestQueue
				jlog.Infof("协程#%03d开始处理%s，队列大小%d", goroutineId, u.username, len(requestQueue))

				startTime := time.Now()
				err := submitJskb(u.username, u.password)
				if err == nil {
					duration := time.Since(startTime)
					meanDuration = meanDuration*0.75 + duration.Seconds()*0.25
				}

				inQueueMutex.Lock()
				delete(inQueue, u.username)
				inQueueMutex.Unlock()
			}
		}(i)
	}

	// POST /api/test 接收username和password，并将提交健康申报表的申请加入等待队列，如果队列已满则此次申请将不会被处理，会提示客户端。
	http.HandleFunc("/api/submit", func(rw http.ResponseWriter, r *http.Request) {
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

		// 检查是否已经在队列里
		inQueueMutex.RLock()
		_, ok := inQueue[username]
		inQueueMutex.RUnlock()
		if ok {
			rw.WriteHeader(429)
			rw.Write([]byte("此用户已经在申请队列中"))
			return
		}

		inQueueMutex.Lock()
		select {
		case requestQueue <- userInfo{username: username, password: password}:
			inQueue[username] = struct{}{}
			waiting := float64(len(inQueue)) * meanDuration
			inQueueMutex.Unlock()
			msg := fmt.Sprintf("已经加入申请队列中，预计需等待%.0f秒后，可查看微信是否有申报成功提示，如果没有，则表示申报可能失败，最可能的原因是密码错误，还有可能是jksb系统下线了（每天凌晨0点后会下线），极小可能是自动识别验证码错误", waiting)
			rw.Write([]byte(msg))
		default:
			inQueueMutex.Unlock()
			rw.WriteHeader(503)
			rw.Write([]byte("请求队列已满，请过几秒或几分钟再尝试"))
		}
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
		rw.Write([]byte("添加账户成功"))
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
		rw.Write([]byte("删除账户成功"))
	})

	// GET /
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			rw.WriteHeader(405)
			rw.Write([]byte("请求非GET方法"))
			return
		}
		rw.Header().Add("Content-Type", "text/html")
		rw.Write(indexPage)
	})
}
