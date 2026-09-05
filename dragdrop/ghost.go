package dragdrop

import "github.com/draxxris/rtgui/core"

// Ghost is a draggable payload's overlay representation.
type Ghost struct {
	// Payload is the application value represented by the ghost.
	Payload Payload
	// Pos is the current logical pointer position.
	Pos core.Vec2
	// Size is the overlay size in logical pixels.
	Size core.Vec2
	// Alpha is the overlay opacity.
	Alpha float32
}

// NewGhost returns the default 32-pixel overlay for payload at pos.
func NewGhost(payload Payload, pos core.Vec2) Ghost {
	return Ghost{Payload: payload, Pos: pos, Size: core.Vec2{X: 32, Y: 32}, Alpha: 0.8}
}

// Bounds reports the ghost's overlay bounds.
func (g Ghost) Bounds() core.Rect {
	return core.Rect{X: g.Pos.X, Y: g.Pos.Y, W: g.Size.X, H: g.Size.Y}
}
