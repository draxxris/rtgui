// Package layout resolves owned widget frames from authored bounds and
// FrameXML-style point relations.
package layout

// AnchorName returns the stable diagnostic spelling of an anchor.
func AnchorName(anchor Anchor) string {
	switch anchor {
	case AnchorTopLeft:
		return "top-left"
	case AnchorTop:
		return "top"
	case AnchorTopRight:
		return "top-right"
	case AnchorLeft:
		return "left"
	case AnchorCenter:
		return "center"
	case AnchorRight:
		return "right"
	case AnchorBottomLeft:
		return "bottom-left"
	case AnchorBottom:
		return "bottom"
	case AnchorBottomRight:
		return "bottom-right"
	default:
		return "unknown"
	}
}

// anchorFactors maps each point to normalized horizontal and vertical factors.
func anchorFactors(anchor Anchor) (x, y float32) {
	switch anchor {
	case AnchorTopLeft:
		return 0, 0
	case AnchorTop:
		return 0.5, 0
	case AnchorTopRight:
		return 1, 0
	case AnchorLeft:
		return 0, 0.5
	case AnchorCenter:
		return 0.5, 0.5
	case AnchorRight:
		return 1, 0.5
	case AnchorBottomLeft:
		return 0, 1
	case AnchorBottom:
		return 0.5, 1
	case AnchorBottomRight:
		return 1, 1
	default:
		return 0, 0
	}
}
