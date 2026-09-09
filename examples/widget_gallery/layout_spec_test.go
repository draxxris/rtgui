package main

import (
	"testing"

	"github.com/draxxris/rtgui/core"
	"github.com/draxxris/rtgui/ui"
)

// TestGalleryLayoutPlacement locks pixel-identical placement at 1280x780.
// Offsets are authored literally in gallerySlots, so this test is the
// guarantee, not arithmetic.
func TestGalleryLayoutPlacement(t *testing.T) {
	u := ui.New(1280, 780)
	newGallery(u)
	want := map[string]core.Rect{
		"leftPanel":               {X: 28, Y: 74, W: 600, H: 682},
		"rightPanel":              {X: 652, Y: 74, W: 600, H: 682},
		"galleryTitle":            {X: 28, Y: 24, W: 600, H: 26},
		"gallerySubtitle":         {X: 30, Y: 51, W: 1200, H: 20},
		"statusLabel":             {X: 30, Y: 762, W: 1220, H: 20},
		"leftTitle":               {X: 52, Y: 89, W: 552, H: 24},
		"primaryButton":           {X: 52, Y: 122, W: 552, H: 42},
		"enableCheckbox":          {X: 52, Y: 186, W: 552, H: 38},
		"textboxCaption":          {X: 52, Y: 227, W: 552, H: 20},
		"inputTextbox":            {X: 52, Y: 250, W: 552, H: 42},
		"dropdownCaption":         {X: 52, Y: 291, W: 552, H: 20},
		"classDropdown":           {X: 52, Y: 314, W: 552, H: 42},
		"sliderCaption":           {X: 52, Y: 355, W: 552, H: 20},
		"valueSlider":             {X: 52, Y: 378, W: 552, H: 42},
		"valueProgress":           {X: 52, Y: 442, W: 552, H: 42},
		"demoPanel":               {X: 52, Y: 514, W: 552, H: 70},
		"panelText":               {X: 66, Y: 544, W: 524, H: 24},
		"tabbarCaption":           {X: 52, Y: 587, W: 552, H: 20},
		"demoTabs":                {X: 52, Y: 610, W: 552, H: 36},
		"chatCaption":             {X: 52, Y: 651, W: 552, H: 20},
		"chatMessage":             {X: 52, Y: 674, W: 552, H: 68},
		"rightTitle":              {X: 676, Y: 89, W: 552, H: 24},
		"demoLabel":               {X: 676, Y: 114, W: 552, H: 32},
		"demoFrame":               {X: 676, Y: 166, W: 552, H: 164},
		"frameCaption":            {X: 694, Y: 186, W: 516, H: 22},
		"frameChildButton":        {X: 700, Y: 240, W: 456, H: 42},
		"scrollCaption":           {X: 688, Y: 334, W: 528, H: 20},
		"scrollPanel":             {X: 676, Y: 356, W: 552, H: 232},
		"marketGraph":             {X: 676, Y: 356, W: 552, H: 232},
		"chatLog":                 {X: 676, Y: 356, W: 552, H: 232},
		"menuHint":                {X: 688, Y: 598, W: 528, H: 20},
		"tooltipHint":             {X: 688, Y: 616, W: 528, H: 20},
		"questButton":             {X: 676, Y: 646, W: 264, H: 36},
		"categoryButton":          {X: 948, Y: 646, W: 256, H: 36},
		"stateSamples":            {X: 676, Y: 690, W: 552, H: 34},
		"questLog":                {X: 430, Y: 230, W: 420, H: 320},
		"questLog/titlebar":       {X: 438, Y: 238, W: 404, H: 32},
		"questLog/title":          {X: 448, Y: 238, W: 362, H: 32},
		"questLog/close":          {X: 814, Y: 242, W: 24, H: 24},
		"questLog/content":        {X: 438, Y: 270, W: 404, H: 272},
		"questText":               {X: 454, Y: 282, W: 372, H: 150},
		"questAccept":             {X: 454, Y: 460, W: 178, H: 44},
		"questDecline":            {X: 648, Y: 460, W: 178, H: 44},
		"categoryWindow":          {X: 490, Y: 170, W: 300, H: 440},
		"categoryWindow/titlebar": {X: 498, Y: 178, W: 284, H: 32},
		"categoryWindow/title":    {X: 508, Y: 178, W: 242, H: 32},
		"categoryWindow/close":    {X: 754, Y: 182, W: 24, H: 24},
		"categoryWindow/content":  {X: 498, Y: 210, W: 284, H: 392},
		"categoryList":            {X: 514, Y: 222, W: 252, H: 308},
		"sellButton":              {X: 514, Y: 542, W: 252, H: 44},
	}
	for name, wantRect := range want {
		w := u.Lookup(name)
		if w == nil {
			t.Fatalf("missing widget %q", name)
		}
		if got := w.Bounds(); got != wantRect {
			t.Errorf("%s = %+v, want %+v", name, got, wantRect)
		}
	}
}
