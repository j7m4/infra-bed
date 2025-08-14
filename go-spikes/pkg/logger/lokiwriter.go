package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LokiWriter is an io.Writer that sends logs to Loki
type LokiWriter struct {
	url        string
	labels     map[string]string
	client     *http.Client
	bufferSize int
	buffer     []lokiLogEntry
	mu         sync.Mutex
	ticker     *time.Ticker
	done       chan bool
	minLevel   zerolog.Level
}

type lokiLogEntry struct {
	timestamp string
	line      string
}

type lokiPushStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type lokiPushRequest struct {
	Streams []lokiPushStream `json:"streams"`
}

// NewLokiWriter creates a new Loki writer
func NewLokiWriter(url string, labels map[string]string) *LokiWriter {
	w := &LokiWriter{
		url:        url,
		labels:     labels,
		client:     &http.Client{Timeout: 5 * time.Second},
		bufferSize: 100,
		buffer:     make([]lokiLogEntry, 0, 100),
		done:       make(chan bool),
		minLevel:   zerolog.InfoLevel,
	}

	// Start background flusher
	w.ticker = time.NewTicker(2 * time.Second)
	go w.flusher()

	return w
}

// SetMinLevel sets the minimum log level to send to Loki
func (w *LokiWriter) SetMinLevel(level zerolog.Level) {
	w.minLevel = level
}

// Write implements io.Writer interface and processes JSON log lines
func (w *LokiWriter) Write(p []byte) (n int, err error) {
	// Parse the JSON to check the log level
	var logData map[string]interface{}
	if err := json.Unmarshal(p, &logData); err != nil {
		// If we can't parse it, skip it
		return len(p), nil
	}

	// Check log level
	if levelStr, ok := logData["level"].(string); ok {
		level, err := zerolog.ParseLevel(levelStr)
		if err == nil && level < w.minLevel {
			// Skip logs below minimum level
			return len(p), nil
		}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Use the full JSON line as is (it already contains all fields)
	entry := lokiLogEntry{
		timestamp: fmt.Sprintf("%d", time.Now().UnixNano()),
		line:      string(bytes.TrimSpace(p)), // Trim newline
	}

	w.buffer = append(w.buffer, entry)

	// Flush if buffer is full
	if len(w.buffer) >= w.bufferSize {
		w.flush()
	}

	return len(p), nil
}

func (w *LokiWriter) flusher() {
	for {
		select {
		case <-w.ticker.C:
			w.mu.Lock()
			if len(w.buffer) > 0 {
				w.flush()
			}
			w.mu.Unlock()
		case <-w.done:
			w.mu.Lock()
			if len(w.buffer) > 0 {
				w.flush()
			}
			w.mu.Unlock()
			return
		}
	}
}

func (w *LokiWriter) flush() {
	if len(w.buffer) == 0 {
		return
	}

	// Create Loki request
	values := make([][]string, 0, len(w.buffer))
	for _, entry := range w.buffer {
		values = append(values, []string{entry.timestamp, entry.line})
	}

	req := lokiPushRequest{
		Streams: []lokiPushStream{
			{
				Stream: w.labels,
				Values: values,
			},
		},
	}

	// Clear buffer
	w.buffer = w.buffer[:0]

	// Send to Loki (in background to avoid blocking)
	go w.send(req)
}

func (w *LokiWriter) send(req lokiPushRequest) {
	data, err := json.Marshal(req)
	if err != nil {
		return
	}

	resp, err := w.client.Post(w.url, "application/json", bytes.NewReader(data))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// Close flushes remaining logs and stops the writer
func (w *LokiWriter) Close() error {
	w.ticker.Stop()
	close(w.done)
	// Give it a moment to flush
	time.Sleep(100 * time.Millisecond)
	return nil
}

// MultiWriter combines console and Loki writers
type MultiWriter struct {
	writers []io.Writer
}

func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

func (m *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range m.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
	}
	return len(p), nil
}