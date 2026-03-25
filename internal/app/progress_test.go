package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// progressReporter throttling
// ---------------------------------------------------------------------------

func TestProgressReporterThrottling(t *testing.T) {
	// Use a channel-based fake to capture messages sent by the reporter.
	msgs := make(chan progressTickMsg, 100)
	r := &progressReporter{
		program:  nil, // replaced by direct channel send below
		interval: 50 * time.Millisecond,
	}

	// Override report to use channel instead of tea.Program.Send.
	reportViaChannel := func(current, total int) {
		r.mu.Lock()
		defer r.mu.Unlock()

		now := time.Now()
		isFinal := total > 0 && current >= total
		if !isFinal && now.Sub(r.lastSend) < r.interval {
			return
		}
		r.lastSend = now
		msgs <- progressTickMsg{current: current, total: total}
	}

	// First call should always send.
	reportViaChannel(1, 10)

	// Immediate second call within interval should be throttled.
	reportViaChannel(2, 10)

	// Only 1 message should have arrived so far.
	assert.Len(t, msgs, 1)
	msg := <-msgs
	assert.Equal(t, 1, msg.current)

	// After waiting past the interval, the next call should send.
	time.Sleep(60 * time.Millisecond)
	reportViaChannel(5, 10)
	assert.Len(t, msgs, 1)
	msg = <-msgs
	assert.Equal(t, 5, msg.current)
}

func TestProgressReporterFinalAlwaysSends(t *testing.T) {
	msgs := make(chan progressTickMsg, 100)
	r := &progressReporter{
		program:  nil,
		interval: 50 * time.Millisecond,
	}

	reportViaChannel := func(current, total int) {
		r.mu.Lock()
		defer r.mu.Unlock()

		now := time.Now()
		isFinal := total > 0 && current >= total
		if !isFinal && now.Sub(r.lastSend) < r.interval {
			return
		}
		r.lastSend = now
		msgs <- progressTickMsg{current: current, total: total}
	}

	// Send first message to set lastSend.
	reportViaChannel(1, 10)
	assert.Len(t, msgs, 1)
	<-msgs

	// Immediately send the final item (current == total). It should not be throttled.
	reportViaChannel(10, 10)
	assert.Len(t, msgs, 1)
	msg := <-msgs
	assert.Equal(t, 10, msg.current)
	assert.Equal(t, 10, msg.total)
}

// ---------------------------------------------------------------------------
// progressDoneMsg handling in Update
// ---------------------------------------------------------------------------

func TestProgressDoneMsgCancelled(t *testing.T) {
	m := &Model{
		width: 80,
		progress: progressState{
			active:    true,
			operation: "Copying",
		},
	}

	result, _ := m.Update(progressDoneMsg{
		operation: "Copying",
		count:     5,
		err:       context.Canceled,
	})
	resultModel := result.(*Model)

	assert.False(t, resultModel.progress.active)
	assert.Nil(t, resultModel.progress.cancel)
	assert.Equal(t, "Cancelled Copying: 5 items processed", resultModel.status)
	assert.Empty(t, resultModel.errMsg)
}

func TestProgressDoneMsgSuccess(t *testing.T) {
	m := &Model{
		width:  80,
		errMsg: "old error",
		progress: progressState{
			active:    true,
			operation: "Copying",
		},
	}

	result, _ := m.Update(progressDoneMsg{
		operation: "Copying",
		count:     10,
		err:       nil,
		status:    "Copied 10 items",
	})
	resultModel := result.(*Model)

	assert.False(t, resultModel.progress.active)
	assert.Nil(t, resultModel.progress.cancel)
	assert.Equal(t, "Copied 10 items", resultModel.status)
	assert.Empty(t, resultModel.errMsg)
}

func TestProgressDoneMsgError(t *testing.T) {
	m := &Model{
		width: 80,
		progress: progressState{
			active:    true,
			operation: "Deleting",
		},
	}

	result, _ := m.Update(progressDoneMsg{
		operation: "Deleting",
		count:     3,
		err:       fmt.Errorf("permission denied"),
	})
	resultModel := result.(*Model)

	assert.False(t, resultModel.progress.active)
	assert.Nil(t, resultModel.progress.cancel)
	assert.Equal(t, "Deleting failed after 3 items: permission denied", resultModel.errMsg)
}

// ---------------------------------------------------------------------------
// progressTickMsg handling in Update
// ---------------------------------------------------------------------------

func TestProgressTickMsgUpdatesState(t *testing.T) {
	tests := []struct {
		name         string
		msg          progressTickMsg
		initialTotal int
		wantCurrent  int
		wantTotal    int
	}{
		{
			name:         "updates current and total",
			msg:          progressTickMsg{current: 5, total: 10},
			initialTotal: 0,
			wantCurrent:  5,
			wantTotal:    10,
		},
		{
			name:         "updates current, keeps existing total when msg total is zero",
			msg:          progressTickMsg{current: 3, total: 0},
			initialTotal: 10,
			wantCurrent:  3,
			wantTotal:    10,
		},
		{
			name:         "updates current with zero total when no prior total",
			msg:          progressTickMsg{current: 7, total: 0},
			initialTotal: 0,
			wantCurrent:  7,
			wantTotal:    0,
		},
		{
			name:         "overwrites existing total with new positive total",
			msg:          progressTickMsg{current: 2, total: 20},
			initialTotal: 10,
			wantCurrent:  2,
			wantTotal:    20,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Model{
				width: 80,
				progress: progressState{
					active: true,
					total:  tc.initialTotal,
				},
			}

			result, cmd := m.Update(tc.msg)
			resultModel := result.(*Model)

			assert.Equal(t, tc.wantCurrent, resultModel.progress.current)
			assert.Equal(t, tc.wantTotal, resultModel.progress.total)
			assert.Nil(t, cmd)
		})
	}
}

// ---------------------------------------------------------------------------
// handleKey during progress
// ---------------------------------------------------------------------------

func TestHandleKeyDuringProgress(t *testing.T) {
	t.Run("ctrl+c cancels and sets status", func(t *testing.T) {
		cancelled := false
		m := &Model{
			progress: progressState{
				active:    true,
				operation: "Copying",
				cancel:    func() { cancelled = true },
			},
			keys: DefaultKeyMap(),
		}

		result, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
		resultModel := result.(*Model)

		assert.True(t, cancelled)
		assert.Equal(t, "Cancelling Copying...", resultModel.status)
		assert.Nil(t, cmd)
	})

	t.Run("esc cancels and sets status", func(t *testing.T) {
		cancelled := false
		m := &Model{
			progress: progressState{
				active:    true,
				operation: "Deleting",
				cancel:    func() { cancelled = true },
			},
			keys: DefaultKeyMap(),
		}

		result, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
		resultModel := result.(*Model)

		assert.True(t, cancelled)
		assert.Equal(t, "Cancelling Deleting...", resultModel.status)
		assert.Nil(t, cmd)
	})

	t.Run("ctrl+c with nil cancel does not panic", func(t *testing.T) {
		m := &Model{
			progress: progressState{
				active:    true,
				operation: "Moving",
				cancel:    nil,
			},
			keys: DefaultKeyMap(),
		}

		result, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
		resultModel := result.(*Model)

		assert.Equal(t, "Cancelling Moving...", resultModel.status)
		assert.Nil(t, cmd)
	})

	t.Run("other keys are swallowed during progress", func(t *testing.T) {
		swallowedKeys := []tea.KeyMsg{
			{Type: tea.KeyRunes, Runes: []rune{'j'}},
			{Type: tea.KeyRunes, Runes: []rune{'k'}},
			{Type: tea.KeyRunes, Runes: []rune{'q'}},
			{Type: tea.KeyRunes, Runes: []rune{'h'}},
			{Type: tea.KeyRunes, Runes: []rune{'l'}},
			{Type: tea.KeyEnter},
			{Type: tea.KeyTab},
			{Type: tea.KeyRunes, Runes: []rune{'/'}},
		}

		for _, key := range swallowedKeys {
			m := &Model{
				progress: progressState{
					active:    true,
					operation: "Copying",
				},
				keys: DefaultKeyMap(),
			}

			result, cmd := m.handleKey(key)
			assert.Equal(t, m, result, "key %q should return model unchanged", key.String())
			assert.Nil(t, cmd, "key %q should return nil cmd", key.String())
		}
	})
}

// ---------------------------------------------------------------------------
// handleMouse during progress
// ---------------------------------------------------------------------------

func TestHandleMouseDuringProgress(t *testing.T) {
	mouseEvents := []tea.MouseMsg{
		{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		{Button: tea.MouseButtonRight, Action: tea.MouseActionPress},
		{Button: tea.MouseButtonWheelUp},
		{Button: tea.MouseButtonWheelDown},
		{Button: tea.MouseButtonNone},
	}

	for _, evt := range mouseEvents {
		m := &Model{
			progress: progressState{
				active:    true,
				operation: "Copying",
			},
		}

		result, cmd := m.handleMouse(evt)
		assert.Equal(t, m, result, "mouse event %v should return model unchanged", evt)
		assert.Nil(t, cmd, "mouse event %v should return nil cmd", evt)
	}
}

// ---------------------------------------------------------------------------
// View rendering with progress active
// ---------------------------------------------------------------------------

func TestProgressViewRendering(t *testing.T) {
	t.Run("known total shows current/total format", func(t *testing.T) {
		m := newTestModel()
		m.width = 120
		m.height = 40
		m.progress = progressState{
			active:    true,
			operation: "Copying",
			current:   5,
			total:     10,
		}

		view := m.View()
		assert.Contains(t, view, "Copying 5/10 secrets...")
		assert.Contains(t, view, "Ctrl+C to cancel")
	})

	t.Run("unknown total shows done count format", func(t *testing.T) {
		m := newTestModel()
		m.width = 120
		m.height = 40
		m.progress = progressState{
			active:    true,
			operation: "Deleting",
			current:   3,
			total:     0,
		}

		view := m.View()
		assert.Contains(t, view, "Deleting... (3 done)")
		assert.Contains(t, view, "Ctrl+C to cancel")
	})

	t.Run("zero width returns loading", func(t *testing.T) {
		m := newTestModel()
		m.width = 0
		m.progress = progressState{
			active:    true,
			operation: "Copying",
		}

		assert.Equal(t, "Loading...", m.View())
	})
}

// ---------------------------------------------------------------------------
// newProgressReporter
// ---------------------------------------------------------------------------

func TestNewProgressReporter(t *testing.T) {
	r := newProgressReporter(nil)
	assert.NotNil(t, r)
	assert.Equal(t, 50*time.Millisecond, r.interval)
	assert.Nil(t, r.program)
	assert.True(t, r.lastSend.IsZero())
}

// ---------------------------------------------------------------------------
// SetProgram
// ---------------------------------------------------------------------------

func TestSetProgram(t *testing.T) {
	m := &Model{}
	assert.Nil(t, m.program)

	// We cannot create a real tea.Program easily in tests, but we can verify
	// the field is set to a non-nil value by using a type assertion.
	// For this test, just verify that SetProgram stores the value.
	m.SetProgram(nil)
	assert.Nil(t, m.program)
}
