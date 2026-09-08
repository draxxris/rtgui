package dragdrop

import (
	"errors"

	"github.com/draxxris/rtgui/core"
)

var (
	errNilController = errors.New("dragdrop: nil controller")
	errNilTarget     = errors.New("dragdrop: nil target")
	errEmptyTarget   = errors.New("dragdrop: empty target name")
)

// DropTarget receives compatible payloads dropped inside Bounds. Name is the
// controller-local registry key.
type DropTarget struct {
	// Name is the controller-local target identity.
	Name string
	// Bounds is the logical area that accepts a pointer.
	Bounds core.Rect
	// Accepts optionally rejects payloads before OnDrop is called.
	Accepts func(Payload) bool
	// OnDrop receives an accepted payload synchronously.
	OnDrop func(Payload)
}

// RegisterTarget adds target to c. Registering an existing name replaces its
// target without changing its original stacking position. Later targets win.
func (c *Controller) RegisterTarget(target *DropTarget) error {
	if c == nil {
		return errNilController
	}
	if target == nil {
		return errNilTarget
	}
	if target.Name == "" {
		return errEmptyTarget
	}
	if c.targets == nil {
		c.targets = make(map[string]*DropTarget)
	}
	if _, exists := c.targets[target.Name]; !exists {
		c.targetOrder = append(c.targetOrder, target.Name)
	}
	c.targets[target.Name] = target
	if c.IsDragging() {
		c.refreshTarget()
	}
	return nil
}

// RemoveTarget removes name from c and reports whether it was registered.
func (c *Controller) RemoveTarget(name string) bool {
	if c == nil || c.targets == nil {
		return false
	}
	if _, exists := c.targets[name]; !exists {
		return false
	}
	delete(c.targets, name)
	for i, targetName := range c.targetOrder {
		if targetName == name {
			copy(c.targetOrder[i:], c.targetOrder[i+1:])
			c.targetOrder[len(c.targetOrder)-1] = ""
			c.targetOrder = c.targetOrder[:len(c.targetOrder)-1]
			break
		}
	}
	if c.IsDragging() {
		c.refreshTarget()
	}
	return true
}

// targetAt returns the last registered target containing pointerPosition.
func (c *Controller) targetAt(pointerPosition core.Vec2) *DropTarget {
	if c.resolver != nil {
		return c.resolver(pointerPosition)
	}
	for i := len(c.targetOrder) - 1; i >= 0; i-- {
		name := c.targetOrder[i]
		target := c.targets[name]
		if target != nil && target.Bounds.Contains(pointerPosition) {
			return target
		}
	}
	return nil
}
