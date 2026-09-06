package render

import (
	"strings"

	"github.com/draxxris/rtgui/core"
)

// Plain tooltip metrics for v1. Text is single-style; callers split rich
// content into plain lines before calling. Wrapping is word-based.
const (
	// TooltipMaxWidth caps tooltip content width in logical pixels.
	TooltipMaxWidth = 280
	// TooltipFontSize is the fixed tooltip text size in logical pixels.
	// Requested ~1.5x nominal (see drawTextInContent) for a true ~13px EM.
	TooltipFontSize = 20
	// TooltipLineHeight is the fixed vertical advance per wrapped line.
	TooltipLineHeight = 24
	// TooltipPadding is the fallback content inset when no skin padding exists.
	TooltipPadding = 8
	// TooltipCursorOffset positions the popup away from the pointer.
	TooltipCursorOffset = 12
	// TooltipCursorDrop positions the popup below the pointer.
	TooltipCursorDrop = 18
)

// TooltipLines wraps text into display lines that fit maxWidth. It splits on
// explicit newlines first, then greedily packs words. Width is estimated
// from font size so bounds work headless without a raylib context.
func TooltipLines(text string, maxWidth, fontSize float32) []string {
	if text == "" {
		return nil
	}
	if maxWidth <= 0 {
		maxWidth = TooltipMaxWidth
	}
	if fontSize <= 0 {
		fontSize = TooltipFontSize
	}
	charWidth := float32(fontSize) * 0.55
	maxChars := int(maxWidth / charWidth)
	if maxChars < 1 {
		maxChars = 1
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		lines = append(lines, wrapParagraph(paragraph, maxChars)...)
	}
	return lines
}

// wrapParagraph packs one newline-delimited paragraph into fitted lines.
func wrapParagraph(paragraph string, maxChars int) []string {
	trimmed := strings.TrimSpace(paragraph)
	if trimmed == "" {
		return []string{""}
	}
	words := strings.Fields(trimmed)
	lines := make([]string, 0, 1)
	current := words[0]
	for _, word := range words[1:] {
		if len([]rune(current))+1+len([]rune(word)) <= maxChars {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	return append(lines, current)
}

// TooltipOuterBounds returns popup outer bounds for text anchored near pos,
// clamped into logical so the tooltip never leaves the viewport. It flips
// above the cursor when it does not fit below.
func TooltipOuterBounds(text string, pos, logical core.Vec2, padding float32) core.Rect {
	lines := TooltipLines(text, TooltipMaxWidth, TooltipFontSize)
	if len(lines) == 0 {
		return core.Rect{}
	}
	if padding <= 0 {
		padding = TooltipPadding
	}
	charWidth := float32(TooltipFontSize) * 0.55
	widest := 0
	for _, line := range lines {
		if n := len([]rune(line)); n > widest {
			widest = n
		}
	}
	width := float32(widest)*charWidth + padding*2
	if width > TooltipMaxWidth+padding*2 {
		width = TooltipMaxWidth + padding*2
	}
	height := float32(len(lines))*TooltipLineHeight + padding*2
	bounds := core.Rect{X: pos.X + TooltipCursorOffset, Y: pos.Y + TooltipCursorDrop, W: width, H: height}
	if logical.X > 0 && bounds.X+bounds.W > logical.X {
		bounds.X = pos.X - bounds.W - TooltipCursorOffset
	}
	if logical.Y > 0 && bounds.Y+bounds.H > logical.Y {
		bounds.Y = pos.Y - bounds.H - TooltipCursorDrop
	}
	if bounds.X < 0 {
		bounds.X = 0
	}
	if bounds.Y < 0 {
		bounds.Y = 0
	}
	return bounds
}
