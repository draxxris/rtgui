package layout

import "rtgui/core"

// Arrange performs one deterministic measure/arrange pass. Children are
// always resolved from the final parent rectangle, which makes moving a frame
// and arranging again produce the same relative result.
func Arrange(node *Node, parentRect core.Rect) {
	resolved := resolveNode(node, parentRect)
	node.Resolved = resolved
	if node.OnResize != nil {
		node.OnResize(node)
	}
	arrangeChildren(node)
}

// ArrangeRoot places the root exactly in the developer-notified viewport and
// then resolves all descendants. It is used by ApplyViewport so a viewport
// resize cannot be held back by an old root authored size.
func ArrangeRoot(node *Node, viewport core.Rect) {
	node.Resolved = viewport
	if node.OnResize != nil {
		node.OnResize(node)
	}
	arrangeChildren(node)
}

func arrangeChildren(node *Node) {
	for _, child := range node.Children {
		Arrange(child, node.Resolved)
	}
}

func resolveNode(n *Node, parent core.Rect) core.Rect {
	size := Measure(n, parent)
	x, y, w, h := resolvePositionAndSize(n, parent, size)
	w, h = clampSize(n, w, h)
	return core.Rect{X: x, Y: y, W: w, H: h}
}

func resolvePositionAndSize(n *Node, parent core.Rect, size core.Vec2) (x, y, w, h float32) {
	x, y = parent.X, parent.Y
	w, h = size.X, size.Y
	stretchX, stretchY := stretchAxes(n)
	if stretchX {
		x = parent.X + n.Offset.X
		w = parent.X + parent.W + n.OpposingOffset.X - x
	} else {
		x = anchorX(n.Anchor, parent, w) + n.Offset.X
	}
	if stretchY {
		y = parent.Y + n.Offset.Y
		h = parent.Y + parent.H + n.OpposingOffset.Y - y
	} else {
		y = anchorY(n.Anchor, parent, h) + n.Offset.Y
	}
	return x, y, w, h
}

func stretchAxes(n *Node) (bool, bool) {
	if !n.HasOpposingAnchor || n.FixedSize != nil || n.RelativeSize != nil {
		return false, false
	}
	start := anchorEdges(n.Anchor)
	end := anchorEdges(n.OpposingAnchor)
	return n.StretchX || (start.x == edgeLeft && end.x == edgeRight),
		n.StretchY || (start.y == edgeTop && end.y == edgeBottom)
}

func clampSize(n *Node, width, height float32) (float32, float32) {
	width, height = clampLimits(n, width, height)
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return width, height
}

func clampLimits(n *Node, width, height float32) (float32, float32) {
	if n.MinSize != nil {
		if width < n.MinSize.X {
			width = n.MinSize.X
		}
		if height < n.MinSize.Y {
			height = n.MinSize.Y
		}
	}
	if n.MaxSize != nil {
		if width > n.MaxSize.X {
			width = n.MaxSize.X
		}
		if height > n.MaxSize.Y {
			height = n.MaxSize.Y
		}
	}
	return width, height
}

type edge int

const (
	edgeLeft edge = iota
	edgeCenter
	edgeRight
	edgeTop
	edgeBottom
)

type anchorPosition struct{ x, y edge }

func anchorEdges(a Anchor) anchorPosition {
	switch a {
	case AnchorTopLeft:
		return anchorPosition{edgeLeft, edgeTop}
	case AnchorTop:
		return anchorPosition{edgeCenter, edgeTop}
	case AnchorTopRight:
		return anchorPosition{edgeRight, edgeTop}
	case AnchorLeft:
		return anchorPosition{edgeLeft, edgeCenter}
	case AnchorCenter:
		return anchorPosition{edgeCenter, edgeCenter}
	case AnchorRight:
		return anchorPosition{edgeRight, edgeCenter}
	case AnchorBottomLeft:
		return anchorPosition{edgeLeft, edgeBottom}
	case AnchorBottom:
		return anchorPosition{edgeCenter, edgeBottom}
	case AnchorBottomRight:
		return anchorPosition{edgeRight, edgeBottom}
	default:
		return anchorPosition{edgeLeft, edgeTop}
	}
}

func anchorX(a Anchor, parent core.Rect, width float32) float32 {
	switch anchorEdges(a).x {
	case edgeCenter:
		return parent.X + (parent.W-width)/2
	case edgeRight:
		return parent.X + parent.W - width
	default:
		return parent.X
	}
}

func anchorY(a Anchor, parent core.Rect, height float32) float32 {
	switch anchorEdges(a).y {
	case edgeCenter:
		return parent.Y + (parent.H-height)/2
	case edgeBottom:
		return parent.Y + parent.H - height
	default:
		return parent.Y
	}
}

// MoveFrame updates the frame's local start offset and all resolved rectangles.
// Child authored rectangles remain local; after a subsequent Arrange they are
// resolved once against the moved parent (not moved twice).
func MoveFrame(n *Node, delta core.Vec2) {
	n.Offset.X += delta.X
	n.Offset.Y += delta.Y
	n.Rect.X += delta.X
	n.Rect.Y += delta.Y
	moveResolved(n, delta)
}

func moveResolved(n *Node, delta core.Vec2) {
	n.Resolved.X += delta.X
	n.Resolved.Y += delta.Y
	for _, child := range n.Children {
		moveResolved(child, delta)
	}
}
