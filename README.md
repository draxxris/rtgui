# rtgui — pure-Go textured GUI

`rtgui` is a small GUI toolkit built on
[`github.com/gen2brain/raylib-go/raylib`](https://github.com/gen2brain/raylib-go).
The module is organized as independent packages instead of one root package.

```text
Go application → widgets / layout / transform → render → raylib-go
```

## Packages

| Package | Responsibility |
| --- | --- |
| `rtgui/core` | Geometry, colors, widget kinds, states, and shared snapshots |
| `rtgui/widgets` | Renderer-independent widget state and interaction |
| `rtgui/input` | Explicit pointer-capture ownership |
| `rtgui/text` | UTF-8-aware text buffers |
| `rtgui/transform` | Physical/logical mapping, snapping, and clipping |
| `rtgui/skin` | Runtime skin descriptors and registry |
| `rtgui/assets` | Optional asset metadata for tooling and layout |
| `rtgui/render` | Themes, draw calls, nine-patch geometry, and raylib drawing |
| `rtgui/layout` | Anchors, measurement, arrangement, and relative movement |
| `rtgui/dragdrop` | Drag state, targets, payloads, and ghosts |
| `rtgui/sim` | Headless harness that simulates human events (`Click`/`Type`) |
| `rtgui/ui` | Application facade: registry, `HandleMouse`/`HandleKey` dispatch, `OnClick`/`OnChange`/`OnText`, `Draw` |

The module root contains no Go package. `render` is the only library package
that imports raylib; the other packages (`ui` included) remain headless and
renderer-neutral.

## Requirements

- Go 1.27 or newer
- A C toolchain and raylib system dependencies for the gallery

On Debian or Ubuntu:

```sh
sudo apt-get install build-essential libgl1-mesa-dev libx11-dev \
  libxcb1-dev libxkbcommon-dev libwayland-dev libasound2-dev pkg-config
```

## Test

```sh
go test ./...
go vet ./...
gofmt -l .
```

The unit tests do not require a display. The `mise.toml` file provides Go 1.27
and the `mise run test` and `mise run vet` shortcuts.

## Example

```sh
go run ./examples/widget_gallery -frames 3 -screenshot out.png
go run ./examples/ui_sample -frames 3
```

For a headless Linux run, use a virtual display:

```sh
xvfb-run --auto-servernum --server-args="-screen 0 1280x800x24" \
  go run ./examples/widget_gallery -frames 3 -screenshot out.png
```

Without a display, the gallery runs a headless smoke path and writes a
placeholder PNG when `-screenshot` is supplied.

## API shape

The application fixes a logical design resolution at startup. Resizing the
window rescales the UI around it instead of reflowing the layout, so
components scale with the window:

```go
viewport := core.Viewport{
    Viewport:    core.Rect{W: 800, H: 600},
    LogicalSize: core.Vec2{X: 800, Y: 600},
}
uiTransform := transform.New(viewport)
capture := input.NewCapture()
theme := render.NewTheme(uiTransform)
button := widgets.NewButton("okButton", core.Rect{W: 120, H: 40}, "OK")

mouse := uiTransform.PhysicalToViewport(core.Vec2{X: 20, Y: 20})
button.UpdateHover(mouse)
button.Press(mouse, capture)
_ = theme.DrawWidget(button.Info(), button.Text, button.Value, button.Checked)
```

Widgets are named with strings. The external `Name` maps to an internal
numeric ID (FNV-1a, deterministic per name) used by `input.Capture` and
`core.WidgetInfo`, so the renderer path is unchanged apart from carrying
the name through.

Human-event simulation stays instance-owned via `rtgui/sim` (no globals):

```go
stage := sim.NewStage()
button := widgets.NewButton("okButton", core.Rect{W: 120, H: 40}, "OK")
_ = stage.Register(button)
stage.Click("okButton")
stage.Type("myTextField", "hello world") // focus-first append
```

`Register` returns an error on duplicate or empty names. `Click` drives
hover/press/release at the widget center and `Type` focuses the textbox
first, then appends per rune (UTF-8 safe); both report `false` on
unknown, disabled, or wrong-kind targets.

`render.Theme` owns its skin registry and draw log and uses the supplied
`transform.Transform` for viewport and pixel-snap state. `input.Capture` and
`transform.Transform` are instance-owned, so separate windows do not share
hidden global UI state.

Most applications should use the `rtgui/ui` facade instead of wiring the
primitives above by hand. Raylib still owns the OS window and frame loop; the
UI reports whether it consumed each polled input so game input keeps working:

```go
u := ui.New(800, 600)
u.Add(widgets.NewButton("primaryButton", core.Rect{X: 40, Y: 40, W: 200, H: 42}, "Primary"))
u.OnClick("primaryButton", func() { status = "Primary clicked" })
u.OnChange("valueSlider", func(v float32) { progress.Value = v })
u.OnText("inputBox", func(s string) { status = "typed: " + s })

physical := rl.GetMousePosition()
mouseHandled := u.HandleMouse(ui.MouseEvent{
    Pos:      u.ToLogical(core.Vec2{X: physical.X, Y: physical.Y}),
    Pressed:  rl.IsMouseButtonPressed(rl.MouseButtonLeft),
    Down:     rl.IsMouseButtonDown(rl.MouseButtonLeft),
    Released: rl.IsMouseButtonReleased(rl.MouseButtonLeft),
    Wheel:    rl.GetMouseWheelMove(),
})
if !mouseHandled {
    // UI did not want it: camera zoom / drag / world picking runs here.
}
keyHandled := u.HandleKey(ui.KeyEvent{Chars: runes, Backspace: bs, Escape: esc})
if !keyHandled {
    // No focused textbox ate the input: game hotkeys run here.
}
u.Draw()
```

`HandleMouse`/`HandleKey` return `handled=true` only when the UI used the
input: press/drag/release on a hit widget (or an active capture), wheel over
a scrolled widget, chars/backspace into a focused textbox, or an `Escape` that
blurred focus. Hover alone, empty-space clicks (which blur but pass through),
and off-widget wheel/keys return `false`. `Add` overwrites duplicates without
changing order (unlike `sim.Register`, which errors); callbacks for unknown
names are stored and fire once the widget is added; `nil` removes a callback.

## Gallery contract

The gallery demonstrates buttons, checkbox state, UTF-8 text editing, a
dropdown popup, slider/progress interaction, scrolling with application-owned
scissor mode, relative frame movement, and widget-state samples. It accepts
`-frames N` and `-screenshot PATH`.

Widget art comes from two checked-in sources: the procedural atlas for panel
fills, tracks, and progress, and the Kenney set under `testdata/skins/kenney/`
for button states (blue line rest, blue border hover, red border press),
checkbox empty/cross icons, the slider handle, the dropdown arrow, and the
grey 8-patch panel ring on frames. File-driven LOOK lives in
`testdata/skins/gallery.css` (`Kind[::part][:pseudo]` selectors with
`border-image-source`, `border-image-slice`, `background-image`, `*-tint`,
and `padding`); it layers over the programmatic aux base, so every key the
file authors wins. Gallery typography uses the Grenze family (SIL OFL,
`testdata/fonts/Grenze-OFL.txt`): `Grenze-Light.ttf` for titles, values,
and widget text, `Grenze-LightItalic.ttf` for captions and the status line.

## Raylib pin

The module pins `github.com/gen2brain/raylib-go/raylib` at `v0.60.1`.
Raylib conversion code is kept at the `render` boundary.
