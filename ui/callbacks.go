package ui

import "github.com/draxxris/rtgui/core"

// OnClick registers a synchronous activation callback for name. Unknown names
// are retained across removal and later registration; nil removes this field.
func (u *UI) OnClick(name string, fn func()) {
	if u == nil || name == "" {
		return
	}
	record := u.callback(name)
	record.onClick = fn
	u.storeCallback(name, record)
}

// OnChange registers a synchronous slider callback for name. It runs only
// after a clamped value really changes; nil removes this field.
func (u *UI) OnChange(name string, fn func(float32)) {
	if u == nil || name == "" {
		return
	}
	record := u.callback(name)
	record.onChange = fn
	u.storeCallback(name, record)
}

// OnText registers a synchronous textbox callback for name. It runs only
// after an edit really changes text; nil removes this field.
func (u *UI) OnText(name string, fn func(string)) {
	if u == nil || name == "" {
		return
	}
	record := u.callback(name)
	record.onText = fn
	u.storeCallback(name, record)
}

func (u *UI) callback(name string) callbackRecord {
	if u.callbacks == nil {
		u.callbacks = make(map[string]callbackRecord)
	}
	return u.callbacks[name]
}

func (u *UI) storeCallback(name string, record callbackRecord) {
	if record.onClick == nil && record.onChange == nil && record.onText == nil && record.onTabSelect == nil && record.onLinkClick == nil && record.onLinkTooltip == nil {
		delete(u.callbacks, name)
		return
	}
	u.callbacks[name] = record
}

func (u *UI) fireOnClick(name string) {
	if u != nil {
		if fn := u.callbacks[name].onClick; fn != nil {
			fn()
		}
	}
}

func (u *UI) fireOnChange(name string, value float32) {
	if u != nil {
		if fn := u.callbacks[name].onChange; fn != nil {
			fn(value)
		}
	}
}

func (u *UI) fireOnText(name, value string) {
	if u != nil {
		if fn := u.callbacks[name].onText; fn != nil {
			fn(value)
		}
	}
}

// OnTabSelect registers a synchronous tab-selection callback for name. It
// runs only after the selected index really changes; nil removes this field.
func (u *UI) OnTabSelect(name string, fn func(int)) {
	if u == nil || name == "" {
		return
	}
	record := u.callback(name)
	record.onTabSelect = fn
	u.storeCallback(name, record)
}

func (u *UI) fireOnTabSelect(name string, index int) {
	if u != nil {
		if fn := u.callbacks[name].onTabSelect; fn != nil {
			fn(index)
		}
	}
}

// OnLinkClick registers a synchronous rich-text link activation callback
// for name. Unknown names are retained across removal and later
// registration; nil removes this field.
func (u *UI) OnLinkClick(name string, fn func(core.Link)) {
	if u == nil || name == "" {
		return
	}
	record := u.callback(name)
	record.onLinkClick = fn
	u.storeCallback(name, record)
}

// OnLinkTooltipRequested registers the hover-text provider consulted once
// per link hover change when the link carries no static tooltip. Returning
// "" shows no tooltip; nil removes this field.
func (u *UI) OnLinkTooltipRequested(name string, fn func(core.Link) string) {
	if u == nil || name == "" {
		return
	}
	record := u.callback(name)
	record.onLinkTooltip = fn
	u.storeCallback(name, record)
}

func (u *UI) fireOnLinkClick(name string, link core.Link) {
	if u != nil {
		if fn := u.callbacks[name].onLinkClick; fn != nil {
			fn(link)
		}
	}
}
