package layout

// Anchor helpers for phase 6

func AnchorName(a Anchor) string {
	switch a {
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
