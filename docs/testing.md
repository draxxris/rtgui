# Testing

## Headless gate

Run checks from the module root:

```sh
gofmt -l $(find . -name '*.go' -type f)
git diff --check
go mod tidy -diff
go list ./...
DISPLAY= WAYLAND_DISPLAY= mise run test
mise run vet
mise run complexity
```

The package tests are designed to run without a display. `render.Theme` skips
raylib draw calls until a window is ready; attach a bounded
`render.DrawRecorder` when a test needs draw diagnostics. The `render` and `ui`
packages still have a direct raylib/native-toolchain requirement even though
most tests are headless.

## Coverage by package

- `core/`: shared geometry, colors, widget kinds, states, and snapshots.
- `widgets/`: widget accessors, layout frames, hit testing, dropdown snapshots,
  checkbox and slider behavior, scrolling, and UTF-8 editing.
- `text/`: private bounded storage, valid and invalid UTF-8, rune-boundary
  truncation, in-place append/backspace, and allocation reuse.
- `transform/`: viewport mapping, physical hit testing, snapping, and clipping.
- `skin/`: CSS selector/property parsing, cascade order, strict errors, and
  exact/normal-state registry lookup.
- `render/`: nine-patch geometry, exact RGBA tinting, fallback drawing, optional
  bounded recording, font-cache cleanup, CSS merge/inheritance, transactional
  texture ownership, and headless loader errors.
- `layout/`: `SetPoint` equations, size limits, ownership and dependency cycles,
  deterministic arrangement, viewport application, movement, and warm-tree
  allocation behavior.
- `dragdrop/`: threshold transitions, accepted/rejected drops, cancellation,
  target replacement/removal, registration order, hover reset, and controller
  isolation.
- `sim/`: semantic activation, focus, UTF-8 text edits, shared callbacks, and
  nil/disabled/unknown target behavior.
- `ui/`: atomic registry operations, UI-owned hover/press/focus state, handled
  input dispatch, callbacks after real mutations, dropdown ownership, resolved
  layout bounds, and recorder frame boundaries.

CSS fake-backend tests use generated temporary images and do not depend on the
ignored Kenney fixtures. The successful gallery CSS path does require the local
ignored files and a graphics context.

## Harness-driven interaction

`sim.Stage` adapts an existing `ui.UI`; it does not duplicate a widget registry
or interaction state. Register widgets and callbacks with the UI first:

```go
u := ui.New(800, 600)
button := widgets.NewButton("okButton", core.Rect{W: 120, H: 40}, "OK")
field := widgets.NewTextbox("myTextField", core.Rect{W: 200, H: 30}, 64)
_ = u.Add(button, field)

stage, err := sim.NewStage(u)
if err != nil {
    panic(err)
}
stage.Click("okButton")
stage.Type("myTextField", "hello world")
```

`Click`, `Type`, and `Focus` use the same UI semantic transitions and
synchronous callbacks as physical input. `Type` focuses the textbox first and
fires a text callback only if at least one rune changes the value. Empty,
invalid, disabled, unknown, wrong-kind, full-buffer, and empty-backspace paths
do not report a mutation.

## Facade-driven interaction

The UI gets first refusal on each polled frame. The application can continue
processing input when the returned handled value is false:

```go
u := ui.New(800, 600)
_ = u.Add(widgets.NewButton("primaryButton", core.Rect{X: 40, Y: 40, W: 200, H: 42}, "Primary"))
u.OnClick("primaryButton", func() { status = "clicked" })
mouseHandled := u.HandleMouse(ui.MouseEvent{Pos: u.ToLogical(p), Pressed: pressed, Down: down, Released: released, Wheel: wheel})
if !mouseHandled {
    cameraZoom(wheel)
}
keyHandled := u.HandleKey(ui.KeyEvent{Chars: runes, Backspace: backspace, Delete: del, Escape: escape,
    Left: left, Right: right, Home: home, End: end, Shift: shift,
    SelectAll: selectAll, Copy: copy, Cut: cut, Paste: paste, Hotkeys: hotkeys})
if !keyHandled {
    playerMove(keys)
}
u.Draw()
```

Focus is dual-slot. `Focus("field")` owns text editing; `FocusFrame("panel")`
owns container glow and scopes `OnHotkey` registrations. Clicking a child
bubbles container focus to its frame; presses outside every frame clear both.
Test it headless:

```go
stage.FocusFrame("demoFrame")
stage.PressHotkey('R') // fires only in scope, case-insensitively
```

Hover alone and empty-space misses pass through. A press, drag, or release on a
hit widget consumes the gesture; a frame-background press sets container focus
and consumes; an active press remains owned until release.
Wheel input is consumed only over a scroll panel. Textbox navigation,
selection, clipboard, and delete keys are consumed only by a focused textbox
when they change caret, selection, clipboard, or text: Left/Right/Home/End move
the caret (Shift extends), Ctrl-A selects all,
Ctrl-C/X/V copy, cut, and paste, Delete removes forward, and typing or
Backspace edits at the caret replacing any selection. Printable typing consumes
even into a full buffer so the host game never observes a rejected character.
Open menus and dropdown popups swallow key intent; `Escape` is consumed
only when it dismisses a menu, tooltip, keyboard focus, or container focus.
Scoped hotkeys fire only when their frame holds container focus and text does
not claim the frame; `AllowWhenEditing:false` suppresses while swallowing so
the game stays silent, and `Consume:false` fires but passes through.
Duplicate `Add` requests are rejected; use `Remove`
before registering a replacement. Callback and hotkey registrations remain available
after widget removal.

## Performance and race checks

The complete validation matrix includes:

```sh
mise exec -- go test -race ./...
mise exec -- go test -shuffle=on -count=20 ./...
mise exec -- staticcheck ./...
mise exec -- go build ./...
mise exec -- govulncheck ./...
mise exec -- go test -run '^$' -bench . -benchmem ./render ./layout ./transform
```

The focused layout benchmark should report zero steady-state allocations after
its dependency cache is warm. The normal draw benchmark disables recording and
should also report zero allocations for its hot operation.

## Gallery smoke

For a real framebuffer screenshot:

```sh
xvfb-run --auto-servernum --server-args="-screen 0 1280x800x24" \
  mise exec -- go run ./examples/widget_gallery -frames 3 -screenshot /tmp/rtgui-test.png
```

Inspect the resulting PNG manually. Confirm that widget states, CSS tint,
nine-patch borders, layout movement, text, slider/progress, and dropdown popup
remain visible. The gallery teardown must unload CSS-owned textures before the
raylib window closes. The gallery owns no borrowed textures.

## Visual references and known fixture limitation

`testdata/golden/` contains references for manual inspection; there is no
pixel-difference gate because drivers and raylib versions vary.

The Kenney PNGs under `testdata/skins/kenney/` are ignored local fixtures. They
are intentionally not changed or added to `.gitignore`; a clean archive lacks
them and therefore cannot run the file-dependent CSS test or gallery skin.
That accepted limitation is not a release gate.

## Pre-push checklist

```sh
gofmt -l $(find . -name '*.go' -type f)
git diff --check
DISPLAY= WAYLAND_DISPLAY= mise run test
mise run vet
mise run build-linux
mise run build-windows
xvfb-run --auto-servernum --server-args="-screen 0 1280x800x24" \
  mise exec -- go run ./examples/widget_gallery -frames 3 -screenshot /tmp/rtgui-check.png
```
