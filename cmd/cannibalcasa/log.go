// Minimal logging to cannibalcasa.log next to the binary (the door has no console).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logMu   sync.Mutex
	logFile *os.File
	logOnce sync.Once
)

func exeDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "."
}

func logf(format string, a ...any) {
	logOnce.Do(func() {
		logFile, _ = os.OpenFile(filepath.Join(exeDir(), "cannibalcasa.log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	})
	if logFile == nil {
		return
	}
	logMu.Lock()
	defer logMu.Unlock()
	fmt.Fprintf(logFile, "%s "+format+"\n",
		append([]any{time.Now().Format("2006-01-02 15:04:05")}, a...)...)
}
