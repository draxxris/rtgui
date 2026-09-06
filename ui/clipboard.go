package ui

import "github.com/draxxris/rtgui/render"

// ClipboardText returns the current clipboard text. Windowed UIs read the
// system clipboard; headless UIs read the in-memory fallback.
func (u *UI) ClipboardText() string {
	if u == nil {
		return ""
	}
	return u.getClipboard()
}

// SetClipboardText stores text for later paste. Windowed UIs mirror into the
// system clipboard and the in-memory fallback; headless UIs keep the fallback.
func (u *UI) SetClipboardText(text string) {
	if u == nil {
		return
	}
	u.setClipboard(text)
}

// getClipboard returns system text when a window exists, else the fallback.
func (u *UI) getClipboard() string {
	if render.IsWindowReady() {
		return render.GetClipboardText()
	}
	return u.clipboard
}

// setClipboard mirrors text into the fallback and the system when ready.
func (u *UI) setClipboard(text string) {
	u.clipboard = text
	if render.IsWindowReady() {
		render.SetClipboardText(text)
	}
}
