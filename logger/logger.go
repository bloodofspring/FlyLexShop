package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warning
	Error
	Fatal
)

type Logger struct {
	level LogLevel
	file  *os.File
}

var (
	instance *Logger
	once     sync.Once
)

func GetLogger() *Logger {
	once.Do(func() {
		// Открываем файл для записи логов
		file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Error opening log file: %v", err)
		}

		instance = &Logger{
			level: Info,
			file:  file,
		}
	})
	return instance
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	// Получаем информацию о месте вызова
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Формируем сообщение
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Создаем структуру для логирования
	logEntry := struct {
		Timestamp string `json:"timestamp"`
		Level     string `json:"level"`
		File      string `json:"file"`
		Line      int    `json:"line"`
		Message   string `json:"message"`
	}{
		Timestamp: timestamp,
		Level:     strings.ToUpper(level.String()),
		File:      file,
		Line:      line,
		Message:   msg,
	}

	// Сериализуем в JSON с отступами
	jsonData, err := json.MarshalIndent(logEntry, "", "    ")
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}

	// Добавляем перенос строки
	jsonData = append(jsonData, '\n')

	// Записываем в файл
	if l.file != nil {
		if _, err := l.file.Write(jsonData); err != nil {
			log.Printf("Error writing to log file: %v", err)
		}
	}

	// Выводим в соответствующий поток
	switch level {
	case Debug, Info:
		fmt.Fprintln(os.Stdout, string(jsonData))
	case Warning, Error:
		fmt.Fprintln(os.Stderr, string(jsonData))
	case Fatal:
		fmt.Fprintln(os.Stderr, string(jsonData))
		os.Exit(1)
	}
}

func (l LogLevel) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warning:
		return "WARNING"
	case Error:
		return "ERROR"
	case Fatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(Debug, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(Info, format, args...)
}

func (l *Logger) Warning(format string, args ...interface{}) {
	l.log(Warning, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(Error, format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(Fatal, format, args...)
}
