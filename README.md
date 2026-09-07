# rtgui — textured Go GUI library

`rtgui` is a small raylib-backed GUI toolkit published as
`github.com/draxxris/rtgui`. It is organized as focused packages rather than one
root package:

```text
Go application → ui / widgets / layout / transform → render → raylib-go
```

## Packages

| Package | Responsibility |
| --- | --- |
| `github.com/draxxris/rtgui/core` | Geometry, colors, widget kinds, states, and renderer snapshots |
| `github.com/draxxris/rtgui/widgets` | Widget domain data and layout-frame accessors |
| `github.com/draxxris/rtgui/text` | Bounded, private UTF-8 text buffers |
| `github.com/draxxris/rtgui/transform` | Physical/logical mapping, snapping, and clipping |
| `github.com/draxxris/rtgui/skin` | Headless skin descriptors, CSS parsing, and registries |
| `github.com/draxxris/rtgui/render` | Raylib themes, draw calls, fonts, nine-patch geometry, and CSS texture ownership |
| `github.com/draxxris/rtgui/layout` | Measurement, `SetPoint` relations, dependency ordering, and arrangement |
| `github.com/draxxris/rtgui/dragdrop` | Instance-owned drag controllers, targets, payloads, and ghosts |
| `github.com/draxxris/rtgui/sim` | Headless semantic adapter for an existing `ui.UI` |
| `github.com/draxxris/rtgui/ui` | Widget registry, interaction ownership, callbacks, input dispatch, and drawing |

`render` directly imports raylib. `ui` owns a concrete `*render.Theme`, so
importing `ui` also brings in the raylib dependency. All UI, rendering, layout,
drag-and-drop, and simulation operations follow a single-owning-goroutine model.

## Requirements

- Go 1.27 or newer
- A native C toolchain and raylib system dependencies for packages that use
  `render` or `ui`, including the gallery
- MinGW-w64 when using the Windows cross-build task

On Debian or Ubuntu:

```sh
sudo apt-get install build-essential libgl1-mesa-dev libx11-dev \
  libxcb1-dev libxkbcommon-dev libwayland-dev libasound2-dev pkg-config
```

The module pins `github.com/gen2brain/raylib-go/raylib` at `v0.60.1`.

## Test

```sh
DISPLAY= WAYLAND_DISPLAY= mise run test
mise run vet
mise run complexity
```

The headless unit tests do not require a display. `go test ./...` is also useful
when running directly through a configured Go toolchain. The complete release
validation additionally runs race testing, shuffle repetition, staticcheck,
`govulncheck`, platform builds, focused benchmarks, and a three-frame gallery
smoke.

## Example

The runnable example is the widget gallery:

```sh
go run ./examples/widget_gallery -frames 3 -screenshot out.png
```

On a headless Linux host, use a virtual display for a real framebuffer image:

```sh
xvfb-run --auto-servernum --server-args="-screen 0 1280x800x24" \
  go run ./examples/widget_gallery -frames 3 -screenshot out.png
```

Without a display, the gallery runs its headless interaction smoke and writes a
standard-library placeholder PNG when `-screenshot` is supplied.

## UI and callbacks

Most applications should use `ui.UI`. It owns the registry, transform, theme,
hover, press, and focus state. Widgets retain domain data and enabled state;
the UI computes the visual state with the priority disabled, pressed, focused,
hovered, then normal.

```go
u := ui.New(800, 600)
button := widgets.NewButton("primaryButton", core.Rect{X: 40, Y: 40, W: 200, H: 42}, "Primary")
if err := u.Add(button); err != nil {
    log.Fatal(err)
}
u.OnClick("primaryButton", func() { status = "clicked" })

mouseHandled := u.HandleMouse(ui.MouseEvent{
    Pos:      u.ToLogical(core.Vec2{X: mouseX, Y: mouseY}),
    Pressed:  pressed,
    Down:     down,
    Released: released,
    Wheel:    wheel,
})
if !mouseHandled {
    // The application may process world input here.
}
keyHandled := u.HandleKey(ui.KeyEvent{Chars: runes, Backspace: backspace, Delete: del, Escape: escape,
    Left: left, Right: right, Home: home, End: end, Shift: shift,
    SelectAll: selectAll, Copy: copy, Cut: cut, Paste: paste, Hotkeys: hotkeys})
if !keyHandled {
    // The host game owns unfocused, unregistered, and non-modal keys here.
}
u.Draw()
```

Focus is dual-slot: keyboard focus (textbox/dropdown) owns text editing while
container focus (active frame) scopes hotkeys and glows via `Frame:focus`.
Click a frame background or a child inside it to activate it; presses
outside every frame move or lose it. Text wins while editing, so typing
`r` never fires a frame-bound `R`. Register scoped actions once:

```go
u.OnHotkey("moveFrame", 'R', ui.HotkeyOpts{Scope: "demoFrame", Consume: true}, func() {
    // Move the frame; runs only while demoFrame is active and no field edits.
})
```

`HandleKey` reports consumption, not mutation: printable typing into a full
buffer still consumes so the game never observes it, modal menus swallow
intent, and `Consume:false` hotkeys fire but pass through.

`Add` rejects nil, empty-name, and duplicate widgets atomically. `Remove` and
`ClearWidgets` clear active interaction owners. Text and slider callbacks run
only after a real mutation; rejected full-buffer characters and empty backspace
are silent. Textboxes show a focus caret, move it with Left/Right/Home/End
(Shift extends the selection), select all with Ctrl-A, and cut/copy/paste
through the system clipboard windowed or an in-memory fallback headless.
Clicking a textbox focuses it and places the caret. `sim.NewStage(u)` adapts
the same activation, focus, text-edit, and callback path for headless tests
without inventing pointer or hover state.

## Drag controllers

Drag-and-drop is instance-owned. Create one `dragdrop.Controller` per UI or
window, register its targets on that controller, and do not rely on package
global state:

```go
controller := dragdrop.NewController(6)
_ = controller.RegisterTarget(&dragdrop.DropTarget{
    Name: "inventory",
    Bounds: core.Rect{X: 20, Y: 20, W: 160, H: 80},
    OnDrop: func(payload dragdrop.Payload) { /* application action */ },
})
controller.Begin(dragdrop.NewPayload("item-1", "item", item), pressPosition)
controller.Move(pointerPosition)
phase := controller.Drop(pointerPosition)
```

## Layout

Each visual widget has one `layout.Node`. `SetPoint` relates a source point to
a target point; a nil target means the ownership parent. One point uses the
preferred size, while two independent points solve the corresponding origin and
size. Contradictory equations, cycles, out-of-tree targets, and point-derived
sizes outside limits return errors.

```go
parent := widgets.NewFrame("panel", core.Rect{W: 400, H: 200})
child := widgets.NewButton("ok", core.Rect{W: 100, H: 36}, "OK")
_ = parent.Frame().AddChild(child.Frame())
if err := child.SetPoint(layout.AnchorTopLeft, nil, layout.AnchorTopRight, core.Vec2{X: 8}); err != nil {
    log.Fatal(err)
}
if err := layout.Arrange(parent.Frame(), core.Rect{}); err != nil {
    log.Fatal(err)
}
```

Use `layout.AnchorName` when a diagnostic or serialized name is needed. A
widget with no arranged parent keeps its authored constructor bounds.

## Drawing, recording, and CSS skins

Normal `Theme` drawing retains no draw-call history. Diagnostics are opt-in and
bounded:

```go
recorder, err := render.NewDrawRecorder(256)
if err != nil {
    log.Fatal(err)
}
u.Theme().SetDrawRecorder(recorder)
u.Draw() // UI.Draw starts the recorder frame
calls := recorder.Calls()
```

Direct `Theme` users call `BeginFrame` at their frame boundary. `Calls` is a
defensive snapshot and `Truncated` reports a full recorder. A zero `core.Color`
is exact transparent black; code that means no tint must set opaque white
explicitly (`core.Color{R: 255, G: 255, B: 255, A: 255}`).

`LoadCSSFile` validates and uploads a complete candidate skin transactionally.
Each source image is uploaded once, CSS-created textures are owned by the theme,
and programmatic textures remain borrowed. Call `UnloadSkin` while the graphics
context is still active, before closing the raylib window. `ClearSkin` also
clears borrowed programmatic descriptors but never unloads their handles.

Font atlases currently contain printable ASCII plus en dash, em dash, bullet,
and ellipsis. UTF-8 storage therefore does not promise glyph coverage for every
Unicode script; callers needing other scripts must provide an appropriate font
and later rasterization strategy.

## Gallery contract and test assets

The gallery demonstrates textured buttons, checkbox state, bounded UTF-8 editing,
a UI-owned dropdown popup, tab-bar selection, chat message with clickable links,
right-click context menus, hover and
pinned tooltips, slider/progress interaction, scrolling, nine-patch
borders, CSS tinting, layout movement, and visual state samples.

The Kenney PNGs under `testdata/skins/kenney/` are ignored local fixtures. They
exist in the accepted current worktree but are not present in a clean
`git archive`; consequently a clean checkout cannot run the CSS tests or the
file-driven gallery skin. This known image limitation is documented evidence,
not a release-validation gate. Do not change the ignored files or `.gitignore`
to hide it.

There is no software license yet. Selecting one remains an unresolved release
item for the repository owner; this project does not add a license by assumption.
