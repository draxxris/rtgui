package render

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/transform"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// PushClip intersects a logical clip with the active clip. PopClip restores it.
// The scissor conversion includes viewport translation as well as scale.
func (t *Theme) PushClip(rect core.Rect) {
	if len(t.clips) > 0 {
		r, ok := transform.Intersect(rect, t.clips[len(t.clips)-1])
		if !ok {
			r = core.Rect{}
		}
		rect = r
	}
	t.clips = append(t.clips, rect)
	t.applyClip(rect)
}

// PopClip restores the enclosing scissor instead of disabling all clipping.
func (t *Theme) PopClip() {
	if len(t.clips) == 0 {
		return
	}
	t.clips = t.clips[:len(t.clips)-1]
	if len(t.clips) > 0 {
		t.applyClip(t.clips[len(t.clips)-1])
		return
	}
	if rl.IsWindowReady() {
		rl.EndScissorMode()
	}
}

// applyClip maps both corners to physical coordinates before integer rounding.
func (t *Theme) applyClip(rect core.Rect) {
	if !rl.IsWindowReady() {
		return
	}
	tr := t.ensureTransform()
	a := tr.ViewportToPhysical(core.Vec2{X: rect.X, Y: rect.Y})
	b := tr.ViewportToPhysical(core.Vec2{X: rect.X + rect.W, Y: rect.Y + rect.H})
	rl.BeginScissorMode(int32(a.X), int32(a.Y), int32(max(0, b.X-a.X)), int32(max(0, b.Y-a.Y)))
}
