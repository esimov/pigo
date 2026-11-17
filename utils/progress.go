package utils

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ProcessIndicator initializes the progress indicator.
type ProcessIndicator struct {
	mu         *sync.RWMutex
	delay      time.Duration
	writer     io.Writer
	message    string
	lastOutput string
	StopMsg    string
	hideCursor bool
	stopChan   chan struct{}
}

const (
	errorColor   = "\x1b[31m"
	successColor = "\x1b[32m"
	defaultColor = "\x1b[0m"
)

// NewIndicator instantiates a new progress indicator.
func NewIndicator(msg string, d time.Duration) *ProcessIndicator {
	return &ProcessIndicator{
		mu:         &sync.RWMutex{},
		delay:      d,
		writer:     os.Stderr,
		message:    msg,
		hideCursor: false,
		stopChan:   make(chan struct{}, 1),
	}
}

// Start starts the progress indicator.
func (pi *ProcessIndicator) Start() {
	if pi.hideCursor && runtime.GOOS != "windows" {
		// hides the cursor
		fmt.Fprintf(pi.writer, "\033[?25l")
	}

	go func() {
		for {
			for _, r := range `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` {
				select {
				case <-pi.stopChan:
					return
				default:
					pi.mu.Lock()

					output := fmt.Sprintf("\r%s%s %c%s", pi.message, successColor, r, defaultColor)
					fmt.Fprintf(pi.writer, "%s", output)
					pi.lastOutput = output

					pi.mu.Unlock()
					time.Sleep(pi.delay)
				}
			}
		}
	}()
}

// Stop stops the progress indicator.
func (pi *ProcessIndicator) Stop() {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	pi.clear()
	pi.RestoreCursor()
	if len(pi.StopMsg) > 0 {
		fmt.Fprintf(pi.writer, "%s", pi.StopMsg)
	}
	pi.stopChan <- struct{}{}
}

// RestoreCursor restores back the cursor visibility.
func (pi *ProcessIndicator) RestoreCursor() {
	if pi.hideCursor && runtime.GOOS != "windows" {
		// makes the cursor visible
		fmt.Fprint(pi.writer, "\033[?25h")
	}
}

// clear deletes the last line. Caller must hold the the locker.
func (pi *ProcessIndicator) clear() {
	n := utf8.RuneCountInString(pi.lastOutput)
	if runtime.GOOS == "windows" {
		clearString := "\r" + strings.Repeat(" ", n) + "\r"
		fmt.Fprint(pi.writer, clearString)
		pi.lastOutput = ""
		return
	}
	for _, c := range []string{"\b", "\127", "\b", "\033[K"} { // "\033[K" for macOS Terminal
		fmt.Fprint(pi.writer, strings.Repeat(c, n))
	}
	fmt.Fprintf(pi.writer, "\r\033[K") // clear line
	pi.lastOutput = ""
}
