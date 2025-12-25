package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

// NewWorkerID builds a unique identifier based on hostname, pid, and random suffix.
func NewWorkerID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "worker"
	}
	pid := os.Getpid()
	return fmt.Sprintf("%s:%d:%s", hostname, pid, randomHex(6))
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(i + 1)
		}
	}
	return hex.EncodeToString(b)
}
