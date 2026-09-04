package layout

import "rtgui/core"

func ApplyViewport(node *Node, vp core.Viewport) {
	// Developer-driven viewport plumbing: root viewport set/notify, per-frame offsets
	ArrangeRoot(node, vp.Viewport)
}
