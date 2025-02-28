package util

import (
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"
)

func Restart(f func(), msg string) {
	log := zap.L()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "recover", r)

			switch t := r.(type) {
			case error:
				log.Error("Recover panic", zap.Error(t))
			case string:
				log.Error("Recover panic", zap.String("error", t))
			}
		}
		time.Sleep(time.Second * 5)
		go Restart(f, msg)
	}()

	log.Info(msg)
	f()
}

func PreventPanic(action string) {
	log := zap.L()
	if r := recover(); r != nil {
		var stack [4096]byte
		runtime.Stack(stack[:], false)
		log.Error(action, zap.String("stack", string(stack[:])))
		log.Error(action, zap.Reflect("recover", r))
		fmt.Println(string(stack[:]))
	}
}
