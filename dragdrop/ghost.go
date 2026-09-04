package dragdrop

import "rtgui/core"

// Ghost rendered in top overlay layer

type Ghost struct {
	Payload Payload
	Pos     core.Vec2
	Size    core.Vec2
	Alpha   float32
}

func NewGhost(payload Payload, pos core.Vec2) Ghost {
	return Ghost{Payload: payload, Pos: pos, Size: core.Vec2{X: 32, Y: 32}, Alpha: 0.8}
}
func (g Ghost) Bounds() core.Rect {
	return core.Rect{X: g.Pos.X, Y: g.Pos.Y, W: g.Size.X, H: g.Size.Y}
}
