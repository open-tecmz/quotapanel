package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger 全局日志记录器，同时写 stdout 和文件。
type Logger struct {
	mu      sync.Mutex
	file    *os.File
	filePath string
}

var globalLogger *Logger
var globalMu sync.Mutex

// Init 初始化全局日志记录器。
// logDir 为日志目录，如果为空则只写 stdout。
func Init(logDir string) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalLogger != nil {
		globalLogger.close()
	}

	l := &Logger{}
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
		name := time.Now().Format("20060102150405") + ".log"
		fp := filepath.Join(logDir, name)
		f, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("创建日志文件失败: %w", err)
		}
		l.file = f
		l.filePath = fp
	}

	globalLogger = l
	return nil
}

// GetLogFilePath 返回当前日志文件路径。
func GetLogFilePath() string {
	globalMu.Lock()
	defer globalMu.Unlock()
	if globalLogger == nil {
		return ""
	}
	return globalLogger.filePath
}

// Info 记录 INFO 级别日志。
func Info(format string, args ...interface{}) {
	write("INFO", format, args...)
}

// Error 记录 ERROR 级别日志。
func Error(format string, args ...interface{}) {
	write("ERROR", format, args...)
}

// Warn 记录 WARN 级别日志。
func Warn(format string, args ...interface{}) {
	write("WARN", format, args...)
}

// Debug 记录 DEBUG 级别日志。
func Debug(format string, args ...interface{}) {
	write("DEBUG", format, args...)
}

func write(level, format string, args ...interface{}) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	line := fmt.Sprintf("%s [%s] %s\n",
		time.Now().Format("2006-01-02 15:04:05.000"),
		level,
		msg,
	)

	// 写 stdout
	fmt.Print(line)

	// 写文件
	globalMu.Lock()
	l := globalLogger
	globalMu.Unlock()

	if l != nil && l.file != nil {
		l.mu.Lock()
		_, _ = io.WriteString(l.file, line)
		l.mu.Unlock()
	}
}

func (l *Logger) close() {
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}
