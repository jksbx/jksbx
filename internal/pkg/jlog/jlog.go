/*
jlog包提供日志函数，日志打到stderr里。
*/
package jlog

import (
	"log"
	"os"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "", log.LstdFlags)
}

func Infof(format string, v ...interface{}) {
	logger.Printf("Info: "+format+"\n", v...)
}

func Infoln(v ...interface{}) {
	w := make([]interface{}, 0, len(v)+1)
	w = append(w, "Info:")
	w = append(w, v)
	logger.Println(w...)
}

func Warnf(format string, v ...interface{}) {
	logger.Printf("Warn: "+format+"\n", v...)
}

func Warnln(v ...interface{}) {
	w := make([]interface{}, 0, len(v)+1)
	w = append(w, "warn:")
	w = append(w, v)
	logger.Println(w...)
}

func Errorf(format string, v ...interface{}) {
	logger.Printf("Error: "+format+"\n", v...)
}

func Errorln(v ...interface{}) {
	w := make([]interface{}, 0, len(v)+1)
	w = append(w, "Error:")
	w = append(w, v)
	logger.Println(w...)
}
