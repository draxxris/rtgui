package layout

import "rtgui/core"

// Node — plain frame container with relative-move semantics.
// Widgets placed within frame move relatively when parent frame moves.

type Anchor int

const (
	AnchorTopLeft Anchor = iota
	AnchorTop
	AnchorTopRight
	AnchorLeft
	AnchorCenter
	AnchorRight
	AnchorBottomLeft
	AnchorBottom
	AnchorBottomRight
)

type Node struct {
	ID                string
	Rect              core.Rect
	Anchor            Anchor
	Offset            core.Vec2 // start-anchor offset in parent coordinates
	OpposingAnchor    Anchor
	HasOpposingAnchor bool
	OpposingOffset    core.Vec2 // end-anchor offset; normally negative for an inset
	StretchX          bool
	StretchY          bool
	FixedSize         *core.Vec2 // nil means stretch via opposing anchors
	MinSize, MaxSize  *core.Vec2
	RelativeSize      *core.Vec2 // optional percentage-like (0..1)
	Children          []*Node
	Parent            *Node
	// Stable IDs for inspection/drag-drop
	StableID uint32
	// Resolved after arrange
	Resolved core.Rect
	// Callbacks
	OnResize func(n *Node)
}

func New(id string, rect core.Rect) *Node {
	return &Node{ID: id, Rect: rect, Anchor: AnchorTopLeft,
		Offset: core.Vec2{X: rect.X, Y: rect.Y}, StableID: hashID(id)}
}
func hashID(s string) uint32 {
	h := uint32(2166136261)
	for _, c := range s {
		h ^= uint32(c)
		h *= 16777619
	}
	if h == 0 {
		h = 1
	}
	return h
}
func (n *Node) AddChild(c *Node) {
	c.Parent = n
	n.Children = append(n.Children, c)
}
func (n *Node) SetAnchor(a Anchor)       { n.Anchor = a }
func (n *Node) SetOffset(v core.Vec2)    { n.Offset = v }
func (n *Node) SetFixedSize(v core.Vec2) { n.FixedSize = &v }

// SetOpposingAnchor enables edge-to-edge stretching. For example, a
// top-left/top-left node with a bottom-right opposing anchor fills its
// parent, subject to offsets and min/max sizes. The ABI has no layout types;
// this retained behavior belongs entirely to Go.
func (n *Node) SetOpposingAnchor(a Anchor) {
	n.OpposingAnchor = a
	n.HasOpposingAnchor = true
}

func (n *Node) ClearOpposingAnchor() { n.HasOpposingAnchor = false }

func (n *Node) SetOpposingOffset(v core.Vec2) { n.OpposingOffset = v }

// SetAnchors is a concise alias useful for responsive panels.
func (n *Node) SetAnchors(start, end Anchor) {
	n.Anchor = start
	n.SetOpposingAnchor(end)
}

// SetStretch permits one-axis stretching when the opposing anchor is not a
// fully edge-to-edge pair. FixedSize remains the fallback for other axes.
func (n *Node) SetStretch(horizontal, vertical bool) {
	n.StretchX, n.StretchY = horizontal, vertical
}
