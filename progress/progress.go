package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

type State interface {
	String() string
}

type Progress struct {
	mu sync.Mutex
	w  io.Writer

	pos    int
	states []State
	ticker *time.Ticker

	lastOutput string
}

func NewProgress(w io.Writer) *Progress {
	p := &Progress{w: w}
	go p.start()
	return p
}

func (p *Progress) stop() bool {
	for _, state := range p.states {
		if spinner, ok := state.(*Spinner); ok {
			spinner.Stop()
		}
	}

	if p.ticker != nil {
		p.ticker.Stop()
		p.ticker = nil
		p.render()
		return true
	}

	return false
}

func (p *Progress) Stop() bool {
	stopped := p.stop()
	if stopped {
		fmt.Fprint(p.w, "\n")
	}
	return stopped
}

func (p *Progress) StopAndClear() bool {
	fmt.Fprint(p.w, "\033[?25l")
	defer fmt.Fprint(p.w, "\033[?25h")

	stopped := p.stop()
	if stopped {
		// clear all progress lines
		for i := 0; i < p.pos; i++ {
			if i > 0 {
				fmt.Fprint(p.w, "\033[A")
			}
			fmt.Fprint(p.w, "\033[2K\033[1G")
		}
	}

	return stopped
}

func (p *Progress) Add(key string, state State) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.states = append(p.states, state)
}

func (p *Progress) render() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Hide cursor only once at start
	if p.lastOutput == "" {
		fmt.Fprint(p.w, "\033[?25l")
	}

	// Move cursor up by number of lines we previously output
	if lines := strings.Count(p.lastOutput, "\n"); lines > 0 {
		fmt.Fprintf(p.w, "\033[%dA", lines)
	}

	// Build new output
	var output strings.Builder
	output.Grow(256)

	for i, state := range p.states {
		if i > 0 {
			output.WriteString("\n")
		}

		// Clear entire line
		output.WriteString("\033[2K")
		// Move cursor to start of line
		output.WriteString("\r")

		output.WriteString(state.String())
	}

	// Write the new output
	fmt.Fprint(p.w, output.String())
	p.lastOutput = output.String()
	p.pos = len(p.states)
}

func (p *Progress) start() {
	p.ticker = time.NewTicker(60 * time.Millisecond)
	for range p.ticker.C {
		p.render()
	}
}
