package ui

import (
	"fmt"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/layout"
	"github.com/draxxris/rtgui/transform"
	"github.com/draxxris/rtgui/widgets"
)

// SetParent changes ownership between registered widgets. Empty parent detaches.
// Bounds remain authored in parent-local coordinates; arrange after reparenting.
func (u *UI) SetParent(childName, parentName string) error {
	child := u.Lookup(childName)
	if child == nil {
		return fmt.Errorf("ui: child %q not found", childName)
	}
	var parent *layout.Node
	if parentName != "" {
		p := u.Lookup(parentName)
		if p == nil {
			return fmt.Errorf("ui: parent %q not found", parentName)
		}
		parent = p.Frame()
	}
	for n := parent; n != nil; n = n.Parent() {
		if n == child.Frame() {
			return layout.ErrOwnershipCycle
		}
	}
	old := child.Frame().Parent()
	if old == parent {
		return nil
	}
	if old != nil {
		old.RemoveChild(child.Frame())
	}
	if parent != nil {
		return parent.AddChild(child.Frame())
	}
	return nil
}

// parentWidget resolves the nearest registered ownership ancestor.
func (u *UI) parentWidget(w widgets.Widget) widgets.Widget {
	for n := w.Frame().Parent(); n != nil; n = n.Parent() {
		if p := u.byNode[n]; p != nil {
			return p
		}
	}
	return nil
}

// available checks registration plus inherited visibility and enabled state.
func (u *UI) available(w widgets.Widget) bool {
	if w == nil || u.Lookup(w.Name()) != w {
		return false
	}
	for ; w != nil; w = u.parentWidget(w) {
		if !w.Enabled() || !w.Visible() {
			return false
		}
	}
	return true
}

// frameFor returns the nearest frame in actual ownership, never by geometry.
func (u *UI) frameFor(w widgets.Widget) widgets.Widget {
	for ; w != nil; w = u.parentWidget(w) {
		if w.Kind() == core.WidgetFrame {
			return w
		}
	}
	return nil
}

// removeTree releases registered descendants without mutating child iteration.
func (u *UI) removeTree(node *layout.Node) {
	for i := 0; i < node.ChildCount(); i++ {
		u.removeTree(node.ChildAt(i))
	}
	w := u.byNode[node]
	if w == nil {
		return
	}
	u.clearReferences(w)
	u.removeDragBindings(w.Name())
	*w.Callbacks() = widgets.Callbacks{}
	w.SetOwner(nil)
	delete(u.widgets, w.Name())
	delete(u.byNode, node)
	w.SetTooltipText("")
	delete(u.richTips, w.Name())
	delete(u.tooltipOpts, w.Name())
	delete(u.richCaches, w.Name())
	u.richTipCache.Invalidate()
	for i := len(u.hotkeys) - 1; i >= 0; i-- {
		if u.hotkeys[i].scope == w.Name() {
			u.RemoveHotkey(u.hotkeys[i].id)
		}
	}
	for i, name := range u.order {
		if name != w.Name() {
			continue
		}
		copy(u.order[i:], u.order[i+1:])
		u.order[len(u.order)-1] = ""
		u.order = u.order[:len(u.order)-1]
		break
	}
}

// childClip returns the visible child area, excluding a scroll panel's track.
func (u *UI) childClip(w widgets.Widget) core.Rect {
	if sp, ok := w.(*widgets.ScrollPanel); ok {
		r := u.scrollContentRect(sp)
		if sp.MaxScroll().Y > 0 {
			r.W = max(0, u.scrollTrackRect(sp).X-r.X)
		}
		return r
	}
	return w.Bounds()
}

// hitSurface follows draw stacking and stops at the first opaque surface.
// Disabled surfaces block underlying input but cannot activate.
func (u *UI) hitSurface(pos core.Vec2) widgets.Widget {
	for i := len(u.order) - 1; i >= 0; i-- {
		w := u.widgets[u.order[i]]
		if w == nil || u.parentWidget(w) != nil {
			continue
		}
		if hit := u.hitNode(w.Frame(), pos); hit != nil {
			return hit
		}
	}
	return nil
}

// hitNode searches children front-to-back inside their parent's visible clip.
func (u *UI) hitNode(node *layout.Node, pos core.Vec2) widgets.Widget {
	w := u.byNode[node]
	if w != nil && !w.Visible() {
		return nil
	}
	inside := w == nil || u.childClip(w).Contains(pos)
	if inside {
		for i := node.ChildCount() - 1; i >= 0; i-- {
			if hit := u.hitNode(node.ChildAt(i), pos); hit != nil {
				return hit
			}
		}
	}
	if w != nil && !w.InputTransparent() && w.HitTest(pos) {
		return w
	}
	return nil
}

// drawNode renders one ownership subtree with nested, intersected clipping.
// Frame borders paint after children so body content draws first and the
// ring finishes on top without a second overlay pass.
func (u *UI) drawNode(node *layout.Node, clip core.Rect, clipped bool) {
	w := u.byNode[node]
	if w != nil && !w.Visible() {
		return
	}
	isFrame := w != nil && w.Kind() == core.WidgetFrame
	if w != nil {
		u.drawOne(w)
	}
	if node.ChildCount() == 0 {
		if isFrame {
			u.drawFrameBorder(w)
		}
		return
	}
	childClip, childClipped := clip, clipped
	if w != nil {
		next := u.childClip(w)
		if clipped {
			var ok bool
			next, ok = transform.Intersect(clip, next)
			if !ok {
				if isFrame {
					u.drawFrameBorder(w)
				}
				return
			}
		}
		childClip, childClipped = next, true
	}
	if childClipped {
		u.theme.PushClip(childClip)
	}
	for i := 0; i < node.ChildCount(); i++ {
		u.drawNode(node.ChildAt(i), childClip, childClipped)
	}
	if childClipped {
		u.theme.PopClip()
	}
	if isFrame {
		u.drawFrameBorder(w)
	}
}

// BringToFront raises a root and its complete subtree as one stacking unit.
func (u *UI) BringToFront(name string) bool {
	w := u.Lookup(name)
	if w == nil {
		return false
	}
	for p := u.parentWidget(w); p != nil; p = u.parentWidget(w) {
		w = p
	}
	for i, entry := range u.order {
		if entry != w.Name() {
			continue
		}
		copy(u.order[i:], u.order[i+1:])
		u.order[len(u.order)-1] = entry
		return true
	}
	return false
}
