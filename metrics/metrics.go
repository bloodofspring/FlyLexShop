package metrics

import (
	"sync"
	"time"
)

type Metrics struct {
	mu sync.Mutex

	// Метрики обработки сообщений
	TotalMessages     int64
	FailedMessages    int64
	ProcessingTime    time.Duration
	MaxProcessingTime time.Duration

	// Метрики горутин
	ActiveGoroutines int
	MaxGoroutines    int

	// Метрики ошибок
	ErrorsByType map[string]int64
}

var (
	instance *Metrics
	once     sync.Once
)

func GetMetrics() *Metrics {
	once.Do(func() {
		instance = &Metrics{
			ErrorsByType: make(map[string]int64),
		}
	})
	return instance
}

func (m *Metrics) RecordMessageProcessing(duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalMessages++
	if !success {
		m.FailedMessages++
	}
	m.ProcessingTime += duration
	if duration > m.MaxProcessingTime {
		m.MaxProcessingTime = duration
	}
}

func (m *Metrics) RecordGoroutineCount(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ActiveGoroutines = count
	if count > m.MaxGoroutines {
		m.MaxGoroutines = count
	}
}

func (m *Metrics) RecordError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ErrorsByType[errorType]++
}

func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	return map[string]interface{}{
		"total_messages":      m.TotalMessages,
		"failed_messages":     m.FailedMessages,
		"avg_processing_time": m.ProcessingTime.Seconds() / float64(m.TotalMessages),
		"max_processing_time": m.MaxProcessingTime.Seconds(),
		"active_goroutines":   m.ActiveGoroutines,
		"max_goroutines":      m.MaxGoroutines,
		"errors_by_type":      m.ErrorsByType,
	}
}
