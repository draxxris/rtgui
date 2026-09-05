package layout

import "github.com/draxxris/rtgui/core"

// ApplyViewport arranges the root exactly in the viewport and returns graph or
// constraint validation errors from the layout pass.
func ApplyViewport(node *Node, viewport core.Viewport) error {
	return ArrangeRoot(node, viewport.Viewport)
}
