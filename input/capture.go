// Package input owns interaction state that is shared by widgets and
// drag-and-drop sessions.
package input

import (
	"errors"
	"sync"
)

// Capture tracks the widget that currently owns pointer events. The zero
// value is ready for use. It tracks the internal numeric widget ID derived
// from the external string name (see widgets hashName); zero means no capture.
type Capture struct {
	mu     sync.RWMutex
	id     uint32
	active bool
}

func NewCapture() *Capture { return &Capture{} }

func (c *Capture) Set(id uint32) error {
	if c == nil {
		return errors.New("input: nil capture")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.id = id
	c.active = true
	return nil
}

func (c *Capture) Release() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.id = 0
	c.active = false
}

func (c *Capture) IsCaptured() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.active
}

func (c *Capture) ID() uint32 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}
