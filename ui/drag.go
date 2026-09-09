package ui

import (
	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/dragdrop"
)

// DragController exposes drag phase, payload preview, and ghost position.
// UI owns Begin, Move, Drop, and cancellation when sources use OnDrag.
func (u *UI) DragController() *dragdrop.Controller {
	if u == nil {
		return nil
	}
	if u.drag == nil {
		u.drag = dragdrop.NewController(6)
		u.drag.SetTargetResolver(u.dropTargetAt)
	}
	return u.drag
}

// OnDrag registers a payload factory for a registered widget. Nil removes it.
// A press below the threshold remains a click. A drag never also activates it.
func (u *UI) OnDrag(name string, source func() dragdrop.Payload) {
	if u.Lookup(name) == nil {
		return
	}
	if source == nil {
		delete(u.dragSources, name)
		return
	}
	if u.dragSources == nil {
		u.dragSources = make(map[string]func() dragdrop.Payload)
	}
	u.dragSources[name] = source
}

// OnDrop binds a target to widget ownership, visibility, clipping, and stacking.
// Nil delivery removes the target. Acceptance predicates must not mutate state.
func (u *UI) OnDrop(name string, accepts func(dragdrop.Payload) bool, deliver func(dragdrop.Payload)) {
	if u.Lookup(name) == nil {
		return
	}
	if deliver == nil {
		delete(u.dropTargets, name)
		return
	}
	if u.dropTargets == nil {
		u.dropTargets = make(map[string]*dragdrop.DropTarget)
	}
	u.dropTargets[name] = &dragdrop.DropTarget{Name: name, Accepts: accepts, OnDrop: deliver}
}

// SetDragGhostDrawer installs optional overlay drawing, below popups and tooltips.
func (u *UI) SetDragGhostDrawer(draw func(dragdrop.Ghost)) { u.dragGhost = draw }

// dropTargetAt selects only the topmost surface or its ownership ancestors.
// Rejection never falls through to an unrelated background window.
func (u *UI) dropTargetAt(pos core.Vec2) *dragdrop.DropTarget {
	for w := u.hitSurface(pos); w != nil; w = u.parentWidget(w) {
		if !u.available(w) {
			return nil
		}
		if target := u.dropTargets[w.Name()]; target != nil {
			target.Bounds = w.Bounds()
			return target
		}
	}
	return nil
}

// beginDrag arms a registered source after ordinary press ownership is resolved.
func (u *UI) beginDrag(pos core.Vec2) {
	w := u.hitSurface(pos)
	if !u.available(w) || u.scrollThumbDragging != nil {
		return
	}
	factory := u.dragSources[w.Name()]
	if factory == nil {
		return
	}
	payload := factory()
	if !u.available(w) {
		return
	}
	u.dragSource = w
	u.DragController().Begin(payload, pos)
}

// routeDrag consumes captured motion and ends the old session before callbacks.
func (u *UI) routeDrag(event MouseEvent) bool {
	if u.dragSource == nil {
		return false
	}
	if !u.available(u.dragSource) {
		u.cancelDrag()
		return false
	}
	if event.Pressed {
		return false
	}
	u.drag.Move(event.Pos)
	if !u.drag.IsDragging() {
		if event.Released {
			u.cancelDrag()
		}
		return false
	}
	u.pressed = nil
	u.disarmLink()
	if event.Released {
		u.dragSource = nil
		u.mouseCaptured = false
		u.drag.Drop(event.Pos)
	}
	return true
}

// cancelDrag releases payload references without invoking delivery callbacks.
func (u *UI) cancelDrag() {
	u.dragSource = nil
	if u.drag != nil {
		u.drag.Cancel()
	}
}

// removeDragBindings disposes all widget-associated drag state and closures.
func (u *UI) removeDragBindings(name string) {
	delete(u.dragSources, name)
	delete(u.dropTargets, name)
	if u.dragSource != nil && u.dragSource.Name() == name {
		u.cancelDrag()
	}
	if u.drag != nil && u.drag.IsDragging() {
		u.drag.Move(u.pointer)
	}
}

// reconcileAuxiliary releases hidden sources and tooltip references on draw too.
func (u *UI) reconcileAuxiliary() {
	if u.dragSource != nil && !u.available(u.dragSource) {
		u.cancelDrag()
		u.pressed = nil
	}
	if u.tipWidget != nil && !u.available(u.tipWidget) {
		u.clearLinkTip()
	}
}

// CancelInput releases interaction on OS focus loss or application cancellation.
// The host must call this when it loses the pointer or keyboard device.
func (u *UI) CancelInput() {
	if u == nil {
		return
	}
	u.cancelDrag()
	u.pressed = nil
	u.setHovered(nil)
	u.scrollThumbDragging = nil
	u.scrollThumbHovered = nil
	u.mouseCaptured = false
	u.clearFocus()
	u.clearActiveFrame()
	u.clearLinkTip()
	u.closeMenuState()
	u.HideTooltip()
}

// capturePress records ownership even for non-activatable blocking surfaces.
func (u *UI) capturePress(event MouseEvent) {
	if !event.Pressed {
		return
	}
	u.mouseCaptured = u.hitSurface(event.Pos) != nil
	if u.mouseCaptured {
		u.beginDrag(event.Pos)
	}
}

// finishMouse retains press ownership through release outside every surface.
func (u *UI) finishMouse(event MouseEvent, handled bool) bool {
	captured := u.mouseCaptured
	if event.Released {
		u.mouseCaptured = false
	}
	return handled || captured && (event.Down || event.Pressed || event.Released)
}
