package widgets

import (
	"github.com/draxxris/rtgui/core"
)

// Slider is an interactive normalized range widget with value from 0 to 1.
type Slider struct {
	base
	value    float32
	format   string
	onChange func(float32)
}

// NewSlider returns an enabled slider with a clamped initial value.
func NewSlider(name string, bounds core.Rect, value float32) *Slider {
	s := &Slider{base: newBase(name, core.WidgetSlider, bounds)}
	s.SetValue(value)
	return s
}

// Value returns the current slider position clamped between 0 and 1.
func (s *Slider) Value() float32 {
	if s == nil {
		return 0
	}
	return s.value
}

// SetValue clamps and stores the slider value, reporting whether it changed.
func (s *Slider) SetValue(value float32) bool {
	if s == nil {
		return false
	}
	if value < 0 {
		value = 0
	} else if value > 1 {
		value = 1
	}
	if s.value == value {
		return false
	}
	s.value = value
	return true
}

// Format returns the format string used for the slider readout.
func (s *Slider) Format() string {
	if s == nil {
		return ""
	}
	return s.format
}

// SetFormat configures the format string used for the slider readout.
func (s *Slider) SetFormat(format string) *Slider {
	if s != nil {
		s.format = format
	}
	return s
}

// OnChange attaches a value mutation callback directly to the slider.
func (s *Slider) OnChange(fn func(float32)) *Slider {
	if s != nil {
		s.onChange = fn
	}
	return s
}

// OnChangeHandler returns the slider's direct value mutation callback.
func (s *Slider) OnChangeHandler() func(float32) {
	if s == nil {
		return nil
	}
	return s.onChange
}

// SetTooltip attaches a hover tooltip string directly to the slider.
func (s *Slider) SetTooltip(text string) *Slider {
	s.base.SetTooltip(text)
	return s
}

// SetTextColor configures an explicit text color for the slider readout.
func (s *Slider) SetTextColor(color core.Color) *Slider {
	s.base.SetTextColor(color)
	return s
}

// SetFontSize sets an explicit font size in pixels for the slider readout.
func (s *Slider) SetFontSize(size float32) *Slider {
	s.base.SetFontSize(size)
	return s
}

// SetItalic configures whether the slider readout uses the italic theme font.
func (s *Slider) SetItalic(italic bool) *Slider {
	s.base.SetItalic(italic)
	return s
}

// SetAlign configures the horizontal text alignment for the slider readout.
func (s *Slider) SetAlign(align core.TextAlign) *Slider {
	s.base.SetAlign(align)
	return s
}

// ProgressBar is a non-interactive normalized progress indicator.
type ProgressBar struct {
	base
	value  float32
	format string
}

// NewProgressBar returns an enabled progress bar with a clamped initial value.
func NewProgressBar(name string, bounds core.Rect, value float32) *ProgressBar {
	p := &ProgressBar{base: newBase(name, core.WidgetProgressBar, bounds)}
	p.SetValue(value)
	return p
}

// Value returns the current progress bar fill clamped between 0 and 1.
func (p *ProgressBar) Value() float32 {
	if p == nil {
		return 0
	}
	return p.value
}

// SetValue clamps and stores the progress bar fill, reporting whether it changed.
func (p *ProgressBar) SetValue(value float32) bool {
	if p == nil {
		return false
	}
	if value < 0 {
		value = 0
	} else if value > 1 {
		value = 1
	}
	if p.value == value {
		return false
	}
	p.value = value
	return true
}

// Format returns the format string used for the progress bar readout.
func (p *ProgressBar) Format() string {
	if p == nil {
		return ""
	}
	return p.format
}

// SetFormat configures the format string used for the progress bar readout.
func (p *ProgressBar) SetFormat(format string) *ProgressBar {
	if p != nil {
		p.format = format
	}
	return p
}

// SetTooltip attaches a hover tooltip string directly to the progress bar.
func (p *ProgressBar) SetTooltip(text string) *ProgressBar {
	p.base.SetTooltip(text)
	return p
}

// SetTextColor configures an explicit text color for the progress bar readout.
func (p *ProgressBar) SetTextColor(color core.Color) *ProgressBar {
	p.base.SetTextColor(color)
	return p
}

// SetFontSize sets an explicit font size in pixels for the progress bar readout.
func (p *ProgressBar) SetFontSize(size float32) *ProgressBar {
	p.base.SetFontSize(size)
	return p
}

// SetItalic configures whether the progress bar readout uses the italic theme font.
func (p *ProgressBar) SetItalic(italic bool) *ProgressBar {
	p.base.SetItalic(italic)
	return p
}

// SetAlign configures the horizontal text alignment for the progress bar readout.
func (p *ProgressBar) SetAlign(align core.TextAlign) *ProgressBar {
	p.base.SetAlign(align)
	return p
}
