package layout

import "rtgui/core"

func Measure(n *Node, available core.Rect) core.Vec2 {
	// An explicit fixed/relative size wins; otherwise use authored size and
	// fill only axes without an authored size.
	w, h := measuredSize(n, available)
	w, h = clampLimits(n, w, h)
	measureChildren(n, w, h)
	return core.Vec2{X: w, Y: h}
}

func measuredSize(n *Node, available core.Rect) (float32, float32) {
	w, h := n.Rect.W, n.Rect.H
	if w <= 0 {
		w = available.W
	}
	if h <= 0 {
		h = available.H
	}
	if n.FixedSize != nil {
		return n.FixedSize.X, n.FixedSize.Y
	}
	if n.RelativeSize != nil {
		return available.W * n.RelativeSize.X, available.H * n.RelativeSize.Y
	}
	return w, h
}

func measureChildren(n *Node, width, height float32) {
	for _, child := range n.Children {
		_ = Measure(child, core.Rect{W: width, H: height})
	}
}
