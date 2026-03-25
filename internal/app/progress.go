package app

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// progressState tracks long-running operation progress.
type progressState struct {
	active    bool               // true while an operation is in progress
	cancel    context.CancelFunc // cancels the running operation
	operation string             // e.g. "Copying", "Deleting", "Moving"
	current   int                // items processed so far
	total     int                // total items (0 = unknown)
}

// progressTickMsg is sent from background goroutines to update the progress display.
type progressTickMsg struct {
	current int
	total   int
}

// progressDoneMsg signals that a long-running operation has completed.
type progressDoneMsg struct {
	operation string
	count     int
	err       error
	status    string // success message
}

// progressReporter sends throttled progress updates to a tea.Program.
// It rate-limits sends to avoid flooding the Update loop.
type progressReporter struct {
	program  *tea.Program
	mu       sync.Mutex
	lastSend time.Time
	interval time.Duration
}

// newProgressReporter creates a reporter that sends updates at most every 50ms.
func newProgressReporter(p *tea.Program) *progressReporter {
	return &progressReporter{
		program:  p,
		interval: 50 * time.Millisecond,
	}
}

// report sends a progress update if enough time has elapsed since the last send,
// or if this is the final item (current == total).
func (r *progressReporter) report(current, total int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	isFinal := total > 0 && current >= total
	if !isFinal && now.Sub(r.lastSend) < r.interval {
		return
	}
	r.lastSend = now
	r.program.Send(progressTickMsg{current: current, total: total})
}

// SetProgram stores the tea.Program reference on the model so background
// goroutines can send progress messages.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}
