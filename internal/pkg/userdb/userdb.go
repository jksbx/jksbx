/*
userdb包实现了一个最简的数据库，存储的是(username, password)的键值对，用Go语言
内建的map来存储。每隔一段时间（需调用方指定具体多久）就自动写盘，以此实现持久化。内存
中的数据库，采用了全局读写锁的机制。
*/
package userdb

import (
	"encoding/gob"
	"io"
	"jksbx/internal/pkg/jlog"
	"os"
	"os/signal"
	"sync"
	"time"
)

var dbFilename string
var userData map[string]string
var userMutex *sync.RWMutex

// Initialize载入存储了用户信息的数据，相当于是恢复上次的状态。
func Initialize(filename string) error {
	dbFilename = filename
	userData = map[string]string{}
	userMutex = &sync.RWMutex{}

	file, err := os.Open(dbFilename)
	if err == nil {
		err = loadUserData(file)
		if err != nil {
			return err
		}
	}

	file, err = os.Create(dbFilename)
	if err != nil {
		return err
	}
	err = dumpUserData(file)
	if err != nil {
		return err
	}

	return nil
}

// AddUser原子地新增一名用户，如果username已经存在，则会覆盖。
func AddUser(username, password string) {
	userMutex.Lock()
	defer userMutex.Unlock()

	userData[username] = password
	jlog.Infof("新增用户%s，目前有%d名", username, len(userData))
}

// DeleteUser原子地删除一名用户。
func DeleteUser(username string) {
	userMutex.Lock()
	defer userMutex.Unlock()

	delete(userData, username)
	jlog.Infof("删除用户%s，目前有%d名", username, len(userData))
}

// CheckUser检查用户密码是否正确。
func CheckUser(username, password string) bool {
	userMutex.RLock()
	defer userMutex.RUnlock()

	pwd, ok := userData[username]
	if !ok {
		return false
	}
	return pwd == password
}

// ExistsUser检查是否存在用户。
func ExistsUser(username string) bool {
	userMutex.RLock()
	defer userMutex.RUnlock()

	_, ok := userData[username]
	return ok
}

// ForEach将handler应用到每一名用户上。
func ForEach(handler func(username, password string)) {
	userMutex.RLock()
	defer userMutex.RUnlock()

	for username, password := range userData {
		handler(username, password)
	}
}

// StartAutoJob起一个协程，来定时写盘，并且侦测<Ctrl-C>来写盘。
func StartAutoJob(duration time.Duration) {
	ticker := time.NewTicker(duration)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				jlog.Infof("开始自动写盘%s", dbFilename)
				file, err := os.Create(dbFilename)
				if err != nil {
					jlog.Errorf("无法打开%s来写数据\n", dbFilename)
					break
				}

				err = dumpUserData(file)
				if err != nil {
					jlog.Errorf("写盘错误：%s\n", err.Error())
					break
				}
			}
		}
	}()

	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt)
	go func() {
		<-sigchan
		done <- struct{}{}
		jlog.Infof("准备退出程序，并写盘%s", dbFilename)
		file, err := os.Create(dbFilename)
		if err != nil {
			jlog.Errorf("无法打开%s来写数据\n", dbFilename)
			os.Exit(0)
		}

		err = dumpUserData(file)
		if err != nil {
			jlog.Errorf("写盘错误：%s\n", err.Error())
			os.Exit(0)
		}

		os.Exit(0)
	}()
}

// loadUserData载入用户数据。
func loadUserData(r io.ReadCloser) error {
	dec := gob.NewDecoder(r)
	userMutex.Lock()
	err := dec.Decode(&userData)
	userMutex.Unlock()
	if err != nil {
		return err
	}
	return r.Close()
}

// dumpUserData把用户数据写入指定Writer。
func dumpUserData(w io.WriteCloser) error {
	enc := gob.NewEncoder(w)
	userMutex.RLock()
	err := enc.Encode(userData)
	userMutex.RUnlock()
	if err != nil {
		return err
	}
	return w.Close()
}
