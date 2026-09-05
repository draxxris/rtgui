package layout

import "github.com/draxxris/rtgui/core"

// Measure returns one node's preferred size without traversing children.
// Invalid size configuration produces the zero value; Arrange reports details.
func Measure(node *Node, available core.Rect) core.Vec2 {
	if node == nil {
		return core.Vec2{}
	}
	width, widthErr := preferredAxisSize(node, available, horizontalAxis)
	height, heightErr := preferredAxisSize(node, available, verticalAxis)
	if widthErr != nil || heightErr != nil {
		return core.Vec2{}
	}
	return core.Vec2{X: width, Y: height}
}
