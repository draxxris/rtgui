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

The module root contains no Go package. `render` is the only library package
that imports raylib; the other packages remain headless and renderer-neutral.

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
```

For a headless Linux run, use a virtual display:

```sh
xvfb-run --auto-servernum --server-args="-screen 0 1280x800x24" \
  go run ./examples/widget_gallery -frames 3 -screenshot out.png
```

Without a display, the gallery runs a headless smoke path and writes a
placeholder PNG when `-screenshot` is supplied.

## API shape

The application owns state explicitly:

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

## Gallery contract

The gallery demonstrates buttons, checkbox state, UTF-8 text editing, a
dropdown popup, slider/progress interaction, scrolling with application-owned
scissor mode, relative frame movement, and widget-state samples. It accepts
`-frames N` and `-screenshot PATH`.

The checked-in button texture is
`testdata/skins/button_rectangle_border.png`. Remaining gallery skin regions
come from a procedural atlas.

## Raylib pin

The module pins `github.com/gen2brain/raylib-go/raylib` at `v0.60.1`.
Raylib conversion code is kept at the `render` boundary.
