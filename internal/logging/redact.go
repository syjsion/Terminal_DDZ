package logging

import (
	"io"
	"strings"
	"sync"
)

type RedactingWriter struct {
	Writer  io.Writer
	Secrets []string
	mu      sync.Mutex
}

func (w *RedactingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	text := string(p)
	for _, secret := range w.Secrets {
		if secret == "" {
			continue
		}
		text = strings.ReplaceAll(text, secret, masked(secret))
	}
	_, err := io.WriteString(w.Writer, text)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func masked(secret string) string {
	if len(secret) <= 8 {
		return "[REDACTED]"
	}
	return secret[:4] + "..." + secret[len(secret)-3:]
}
