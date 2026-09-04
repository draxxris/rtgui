package ui

// OnClick registers a click callback for the named widget.
// The callback fires synchronously inside HandleMouse when Release reports a
// real click, after the checkbox toggle so observers see post-toggle state.
// Registering for an unknown name is allowed and fires if the widget is added
// later. A nil fn removes the registration.
func (u *UI) OnClick(name string, fn func()) {
	if u == nil || name == "" {
		return
	}
	if u.onClick == nil {
		u.onClick = make(map[string]func())
	}
	if fn == nil {
		delete(u.onClick, name)
		return
	}
	u.onClick[name] = fn
}

// OnChange registers a slider-change callback for the named widget.
// It fires synchronously inside HandleMouse with the clamped 0..1 value after
// each press or drag that moved the slider. Unknown names are stored; nil
// removes the registration.
func (u *UI) OnChange(name string, fn func(float32)) {
	if u == nil || name == "" {
		return
	}
	if u.onChange == nil {
		u.onChange = make(map[string]func(float32))
	}
	if fn == nil {
		delete(u.onChange, name)
		return
	}
	u.onChange[name] = fn
}

// OnText registers a textbox callback for the named widget.
// It fires synchronously inside HandleKey with the full buffer string after
// each handled edit. Unknown names are stored; nil removes the registration.
func (u *UI) OnText(name string, fn func(string)) {
	if u == nil || name == "" {
		return
	}
	if u.onText == nil {
		u.onText = make(map[string]func(string))
	}
	if fn == nil {
		delete(u.onText, name)
		return
	}
	u.onText[name] = fn
}

// fireOnClick invokes the click callback, if any. State is already consistent;
// a panicking callback propagates but pressed/focus/capture were settled first.
func (u *UI) fireOnClick(name string) {
	if u == nil || name == "" {
		return
	}
	if fn := u.onClick[name]; fn != nil {
		fn()
	}
}

// fireOnChange invokes the slider callback, if any, with the current value.
func (u *UI) fireOnChange(name string, value float32) {
	if u == nil || name == "" {
		return
	}
	if fn := u.onChange[name]; fn != nil {
		fn(value)
	}
}

// fireOnText invokes the textbox callback, if any, with the buffer string.
func (u *UI) fireOnText(name, text string) {
	if u == nil || name == "" {
		return
	}
	if fn := u.onText[name]; fn != nil {
		fn(text)
	}
}
