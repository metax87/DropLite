package logging

import (
	"log"
	"os"
)

// New 创建一个基础日志器，后续可替换为结构化日志实现。
func New() *log.Logger {
	logger := log.New(os.Stdout, "droplite ", log.LstdFlags|log.Lshortfile)
	return logger
}
