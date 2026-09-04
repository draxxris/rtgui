# Testing

## Headless gate

Run the library checks from the module root:

```sh
gofmt -l .
go vet ./...
DISPLAY= WAYLAND_DISPLAY= go test ./...
```

The tests in `core`, `widgets`, `input`, `text`, `transform`, `skin`,
`render`, `layout`, `dragdrop`, and `sim` do not require a display. `render.Theme`
records draw calls and skips raylib drawing when no window is ready.

### Coverage by package

- `widgets/`: hover, press/release, focus, checkbox and slider behavior,
  scrolling, and UTF-8 editing.
- `text/`: NUL-terminated buffer capacity and UTF-8-safe truncation.
- `input/`: explicit pointer-capture lifecycle.
- `transform/`: viewport mapping, physical hit testing, snapping, and clipping.
- `skin/`: exact and normal-state-fallback registry lookup.
- `render/`: nine-patch geometry, content insets, fallback drawing, theme
  isolation, draw logging, sliders, and progress bars.
- `layout/`: anchors, measurement, viewport arrangement, and `MoveFrame`.
- `dragdrop/`: threshold transitions, target ordering, accepted/rejected drops,
  cancellation, and ghosts.
- `sim/`: string-ID registry, center `Click`, and focus-first-append `Type`,
  including duplicate/empty-name, unknown-ID, disabled, and wrong-kind paths.

Optional benchmarks:

```sh
go test -bench . ./...
```

## Harness-driven interaction

Use `rtgui/sim` to drive widgets headlessly without a display or raylib.
Each `Stage` owns its registry plus capture, so tests stay parallel-safe
with no package-global state:

```go
stage := sim.NewStage()
button := widgets.NewButton("okButton", core.Rect{W: 120, H: 40}, "OK")
field := widgets.NewTextbox("myTextField", core.Rect{W: 200, H: 30}, 64)
_ = stage.Register(button)
_ = stage.Register(field)
stage.Click("okButton")
stage.Type("myTextField", "hello world") // focus-first append
```

Semantics: `Click` presses and releases at the widget center and always
leaves capture released; `Type` focuses the textbox first (blurring the
previous one) and appends each rune via `TypeChar`, keeping multi-byte
UTF-8 safe. Both return `false` on unknown, disabled, or wrong-kind
targets instead of failing loudly. `Register` returns an error on
duplicate or empty names without clobbering the original entry. Pair the
harness with `theme.DrawWidget(w.Info(), ...)` to prove the renderer path
still works with string IDs.

## Gallery smoke

With a display:

```sh
go run ./examples/widget_gallery -frames 3 -screenshot out.png
```

On headless Linux:

```sh
xvfb-run --auto-servernum --server-args="-screen 0 1280x800x24" \
  go run ./examples/widget_gallery -frames 3 -screenshot out.png
```

The gallery uses raylib only after a window is ready. Without a display it
runs its headless smoke path and writes a standard-library placeholder PNG.

## Visual references

`testdata/golden/` contains reference images for manual inspection. There is
no automated pixel-difference gate because output can vary across drivers and
raylib versions. Explain any intentional visual change before replacing a
reference image.

## Pre-push checklist

```sh
gofmt -l .
go vet ./...
DISPLAY= WAYLAND_DISPLAY= go test ./...
xvfb-run --auto-servernum go run ./examples/widget_gallery \
  -frames 3 -screenshot /tmp/rtgui_check.png
```
