# Code Audit

> This document retains the original audit findings and their evidence. The
> final public-surface review at the end records which findings were resolved,
> which identifiers remain intentionally valid, and which release decision is
> still owned by the repository owner.

## Verdict

`rtgui` has a good prototype core. It is not ready for use as a public library.

The geometry, headless rendering, and basic event code are easy to read. The test suite
is also substantial for the repository size.

The larger design has too many public concepts and too many sources of state. Several
advertised features do not connect to the main `ui` path.

The next work must reduce the public surface instead of adding features. This repository
can become much smaller without losing its demonstrated behavior.

## Scope

This audit covers all tracked Go source files and all packages. It also covers the
README, test guidance, module files, gallery, fixtures, and recent history.

The library contains approximately 3,577 production lines, excluding the gallery. It
exposes 12 packages and approximately 180 type, function, and method declarations.

The audit found no release tags and no configured Git remote. The repository owner has
now confirmed the intended roles of the isolated packages.

The accepted direction is as follows:

- Remove numeric IDs and their hash functions.
- Remove `assets`.
- Make the UI the sole owner of interaction state.
- Make draw recording explicit and bounded.
- Give CSS textures an owner and an atomic replacement path.
- Keep `dragdrop` as an external library feature, but replace its global state.
- Keep `sim` for RPC-controlled tests.
- Keep `layout` as active work, including `layout.AnchorName`.
- Simplify widgets, CSS tinting, layout, text storage, zero values, and error APIs.
- Keep the direct dependency between `ui`, `render`, and raylib.

## Good parts

- `go vet ./...` passes.
- `go test -race ./...` passes in the current worktree.
- `mise run complexity` passes. The highest production cyclomatic complexity is 14.
- The average reported cyclomatic complexity is 3.91.
- `govulncheck ./...` reports no known vulnerabilities.
- Most package dependencies point in a clear direction.
- Only `render` directly imports raylib among library packages.
- The headless draw path gives tests useful render records.
- The geometry functions are small and deterministic.
- The code uses no reflection, unsafe code, or hidden goroutines.

These strengths make cleanup practical. The repository does not require a full rewrite.

## Critical findings

### A clean checkout fails its tests

The local test suite passes because ignored Kenney image files exist in the worktree.

The `*.png` rule ignores nested skin files. The exceptions only cover PNG files directly
under `testdata/skins` (`.gitignore:16-19`).

No file under `testdata/skins/kenney/` is tracked. However, `render/css_test.go:69-84`
requires two of those files.

The gallery CSS also requires the ignored files (`testdata/skins/gallery.css:7-64`). The
README incorrectly calls these files checked-in assets (`README.md:163-167`).

A test from a clean `git archive` fails with this error:

```text
render: css image "kenney/blue/button_rectangle_line.png":
open ../testdata/skins/kenney/blue/button_rectangle_line.png: no such file or directory
```

This is a known release limitation. A clean clone cannot run the documented test suite.

The repository owner accepts this limitation for the current work. The cleanup must not
change the ignored Kenney files, their tests, or the `.gitignore` rules.

### The draw log grows forever

Every draw operation appends a `DrawCall` (`render/draw.go:61-76`). Logging is always
active.

Neither `UI.Draw` nor the gallery clears this log (`ui/draw.go:14-21`,
`examples/widget_gallery/main.go:748-780`). The gallery runs this path at 60 frames per
second.

The existing benchmark repeatedly appends to the same log
(`render/bench_test.go:18-20`). It reports 414 bytes per operation for one widget part.

This design retains all frame history. A long-running application will retain hundreds
of megabytes of draw records.

`DrawLog` also copies the complete retained history (`render/theme.go:263-268`). This
operation becomes more expensive after every frame.

Accepted action:

- Make recording opt-in.
- Store only the current frame when a recorder is attached.
- Use a bounded buffer if historical records are necessary.
- Keep a dedicated recorder for headless tests.

### CSS textures have no owner or cleanup path

`LoadCSSFile` uploads textures at `render/css.go:197-225`. `Theme` does not record which
textures it owns.

`ClearSkin` only clears a map (`render/theme.go:64-68`). `UnloadFonts` does not unload
skin textures.

Repeated CSS loads upload duplicate textures and abandon the old handles. A long-running
theme reload will leak GPU resources.

The loader also claims atomic application. The implementation writes each descriptor
during the upload loop (`render/css.go:65-76`).

If a later upload fails, earlier descriptors and textures remain. The operation is not
atomic across upload failures.

Accepted action:

- Add explicit ownership for CSS-created textures.
- Build a complete replacement skin before registry publication.
- Unload all new textures after a failed build.
- Swap registries only after every upload succeeds.
- Add `UnloadSkin` for owned resources.
- Keep programmatic textures marked as borrowed.

### UI state has multiple owners

The interaction state exists in several forms:

- `Widget.State`
- `Widget.Focused`
- `UI.focused`
- `UI.pressed`
- `input.Capture`
- `Widget.Enabled` and `StateDisabled`

These values can disagree.

For example, `updateHover` skips the focused widget (`ui/input.go:64-68`).
`refreshDisabled` can then set its state to disabled (`ui/input.go:81-89`).

If the caller enables that widget again, the state can remain disabled. Keyboard input
still reaches the focused widget.

Duplicate registration creates another error. `UI.Add` replaces the map entry but does
not update `focused` or `pressed` (`ui/ui.go:83-102`).

After replacement, keyboard input can mutate an invisible old widget. A release can also
complete a gesture against that old widget.

This behavior conflicts with the documented claim that duplicate adds support layout
rebuilds.

Accepted action:

- Let the widget own its enabled flag.
- Let the UI own focus, press, and hover.
- Compute the visual state from that interaction state.
- Reject duplicate registration and require explicit removal.
- Add explicit `Remove` and `Clear` operations.
- Add tests for removal during focus and press.

## High-value simplification candidates

### Remove the numeric capture identity system

The UI already routes drag and release operations through `UI.pressed`. It does not use
the capture ID for event routing.

`WidgetInfo.HasCapture` is written in `ui/draw.go:33`. The renderer never reads it.

The numeric system creates these obligations:

- `Widget.Name` and `Widget.ID`
- `Capture.id` and `Capture.active`
- Three private hash implementations
- Collision behavior
- Capture mutexes
- Public capture accessors
- Capture fields in render snapshots

The three hash functions are in `widgets/widget.go:33`, `layout/node.go:49`, and
`dragdrop/state.go:48`.

The layout version hashes Unicode runes. The other versions hash UTF-8 bytes. The same
non-ASCII name produces different IDs.

The design also accepts normal 32-bit hash collisions. Two different names can receive
the same capture ID.

`Capture.Set` overwrites any existing owner (`input/capture.go:21-29`).
`Capture.Release` releases every owner without an identity test
(`input/capture.go:32-40`).

The current type therefore does not enforce capture ownership.

Accepted action:

- Remove numeric IDs and all three hash functions.
- Let the UI keep the active widget pointer.
- Let the drag controller keep the active source directly.
- Remove the `input.Capture` package if no other input primitive remains.
- Remove capture data from render snapshots.

This cut breaks public APIs. The repository owner has accepted that break before the
first tagged release.

### Remove `assets` and define the remaining package roles

The original reachability result treated each isolated package as a removal candidate.
The intended external contracts now resolve that uncertainty.

- `assets` has no required consumer and can be removed.
- `dragdrop` is a supported external capability and must remain.
- `sim` supports RPC-controlled tests and intentionally bypasses physical event
  dispatch.
- `layout` is active work and must remain.

The `deadcode` analyzer marks the complete `assets`, `dragdrop`, and `sim` APIs
unreachable from the only shipped executable.

This result only describes the gallery. It does not override the confirmed external
contracts.

Accepted action:

- Remove `assets`, its tests, and its documentation.
- Keep `dragdrop`, but give each UI or window its own controller.
- Keep `sim` separate from production input dispatch.
- Reuse UI state transitions and callback semantics from `sim`.
- Connect resolved layout bounds to widgets without gallery-owned copying.

### Replace the drag-and-drop global registry

`dragdrop/target.go:12-15` stores all targets in package-global maps and slices. The
package has no mutex and no instance boundary.

Multiple windows share these targets. Concurrent registration and hit tests can race.
Forgotten targets retain callbacks and application data.

The drag state also contains redundant or unused values:

- `CurrentPos` and `GhostPos` always move together.
- `SourceID` duplicates `Payload.ID`.
- `Captured` mirrors `Capture`.
- `LongPress` has no reader.
- `Cancel(reason)` discards `reason`.

Accepted action:

Use an instance-owned drag controller. Keep one source identity, one position, and one
capture state. Give the controller explicit target registration and removal operations.

The controller belongs to one UI or window. This boundary corrects target lifetime even
when the application uses only one goroutine.

### Keep `sim` separate, but remove its duplicate state machine

RPC-controlled UI tests are a valid contract. They do not need raylib input or physical
mouse coordinates.

The current `sim.Stage` still duplicates widget registration, focus, press, and typing
rules. Those copies can diverge from normal UI behavior.

Do not move the RPC transport into `ui`. Keep `sim` as the test-facing adapter.

Add semantic UI operations for activation and text edits. Let input dispatch and `sim`
call the same state-transition functions.

This design lets `sim` bypass physical event dispatch. It does not bypass widget
invariants, callbacks, or enabled-state checks.

### Reduce the tagged `Widget` data bag

`widgets.Widget` exposes every field for every widget kind (`widgets/widget.go:12-27`).
Most combinations are invalid.

A label can have a text buffer. A frame can have dropdown items. A progress bar can
become focused.

Kind-specific rules then appear across `widgets`, `ui`, `render`, and `skin`. This
spreads one widget policy across four packages.

The exported fields also let callers change `Name`, `ID`, `Kind`, and `TextBuf` after
registration. These changes can invalidate registry state.

Accepted action:

- Keep identity and kind immutable.
- Hide fields that carry interaction state.
- Put kind traits in one location.
- Use small kind-specific payloads or concrete widget types.
- Keep the renderer snapshot as a value type.

Concrete widget interfaces are not mandatory. A compact tagged representation can remain
if it enforces its invariants.

### Remove CPU tint baking from CSS loading

CSS support uses 595 production lines. This is approximately 17 percent of the library.

The loader decodes an image, applies a CPU tint, encodes PNG data, decodes it through
raylib, and uploads it (`render/css.go:195-225`).

The renderer already passes a tint to `DrawTexturePro` (`render/draw.go:112-113`). It
can share one texture across multiple tint descriptors.

The current path uploads one texture for each file-and-tint pair. It increases CPU work,
GPU memory, code size, and cleanup work.

Accepted action:

- Upload each source image once.
- Store the CSS tint in `SkinDescriptor.Tint`.
- Remove `TintImage`, PNG re-encoding, and tint-specific cache keys.
- Keep alpha multiplication in the draw path.

The parser dependency also weakens the strict-error claim. Its `Parse` function returns
no error and documents poor error handling.

Unsupported parser input can disappear before `skin.ParseCSS` sees a rule. Some invalid
CSS can therefore return no library error.

### Delete public fossils and exact aliases

The following items have no production reader or caller in this repository:

- `core.WidgetInfo.IsClipped`
- `core.WidgetInfo.ClipRect`
- `dragdrop.DragState.LongPress`
- `layout.Node.StableID`
- `skin.NinePatch.Layout`
- `skin.SkinDescriptor.TileMode`
- `skin.PartHighlight`
- `render.DrawMode`
- `render.NinePatchConfig.Mode`
- `render.DebugInfo.PatchBorders`
- `render.ApplyPadding`
- `render.EnforceMinSize`
- `render.DebugBounds`
- `render.EstimateDrawCalls`

`ApplyPadding` is an exact alias for `ContentRect` (`render/geometry.go:114-116`).

`NinePatchRects` accepts a source rectangle and a `NinePatchConfig.Source`. It uses
neither source value (`render/geometry.go:26`).

`EstimateDrawCalls` compares source rectangles instead of texture IDs
(`render/draw.go:413-425`). It cannot measure texture switches correctly.

Accepted action:

Remove these items before a stable release. Keep an item only when a named consumer or
compatibility contract exists.

`layout.AnchorName` is the explicit exception. It remains useful for diagnostics,
serialization, and a future FrameXML-style parser.

### Replace special-case layout fields with point constraints

The proposed FrameXML model is a better direction than the current parent-relative
anchor model.

The current `Anchor` field uses the same point on the child and its parent. The
`OpposingAnchor` and stretch flags then add special cases for common layouts.

A point relation expresses the model directly:

```go
button.SetPoint(
    layout.AnchorTopLeft,
    parentFrame,
    layout.AnchorTopRight,
    core.Vec2{X: 8},
)
```

Use `SetPoint` rather than `SetAnchor` for the Go API. The name makes the two-point
relation clear and matches the FrameXML operation.

One point positions a frame that has a known or measured size. Two independent points
can infer width, height, or both.

This model removes `OpposingAnchor`, `HasOpposingAnchor`, `StretchX`, and `StretchY`.
Top-left and bottom-right point relations already express stretching.

Keep ownership separate from constraints. A parent owns a child, but an anchor target
can be a parent, sibling, or another frame.

Keep the layout algorithm in `layout`. Give each visual widget one frame, and let `ui`
use the resolved frame for hit tests and rendering.

Cross-frame references form a dependency graph. The resolver must order dependencies
and reject cycles with a useful error.

Build and validate the dependency order when constraints change. Reuse that order during
steady-state layout passes.

Use the `Anchor` enum in Go. Use `AnchorName` only at text, diagnostic, or serialization
boundaries.

`Measure` recursively measures every child and discards every result
(`layout/measure.go:5-11`, `layout/measure.go:31-34`).

`Arrange` then measures each child again. A deep tree receives repeated work with no
stored result.

`Node.Rect.X`, `Node.Rect.Y`, and `Node.Offset` also represent local position.
`SetOffset` changes one value, while `MoveFrame` changes both.

The parent pointer is written but never read in the package. Cycles and duplicate
parents are not rejected.

Staticcheck also reports unused initial assignments to `x` and `y` at
`layout/arrange.go:42`.

Accepted action:

- Remove the discarded child measurement.
- Keep one authored position value.
- Keep `Parent` for ownership and the default anchor target.
- Reject ownership and anchor-dependency cycles.
- Define conflict rules for fixed, relative, minimum, maximum, and stretch sizes.
- Make the UI consume resolved bounds without a manual gallery bridge.
- Do not add a general constraint solver. Two point equations per axis are sufficient.

### Replace the text buffer internals

`text.Buffer` stores both `Buf` and `Cap`. Callers can change either field and break the
invariant (`text/buffer.go:7-10`).

`Bytes` returns the mutable internal slice (`text/buffer.go:44`). The caller can remove
the terminator or insert invalid UTF-8.

`Set` allocates a new slice for every edit (`text/buffer.go:34-41`). `Widget.TypeChar`
also creates multiple strings for each rune.

The buffer only repairs UTF-8 after truncation. It accepts invalid UTF-8 when the input
already fits.

No C boundary in this repository uses the NUL terminator.

Accepted action:

- Use the slice length or capacity as the single limit.
- Keep the byte slice private.
- Append and delete runes in place.
- Return whether an edit changed the value.
- Remove NUL termination unless an actual boundary requires it.

This change also corrects callback semantics. `UI.handleText` currently reports a
mutation even when a full buffer rejects a character.

It also reports a mutation for backspace on an empty buffer (`ui/input.go:173-188`).

## Other bad patterns

### Document a single-owner execution model

`UI`, `Theme`, layout, drag-and-drop, and `sim` can use one owning goroutine.

The confirmed contract never operates `sim` and user-driven UI control at the same
time. No synchronization between those paths is necessary.

The current mutexes do not protect complete widget operations. Remove them unless a
specific external contract requires concurrent access.

The package-global drag target registry remains invalid. Its main defects are ownership,
target lifetime, and support for multiple UI instances, not thread safety.

### Zero values act as hidden sentinels

`Alpha == 0` means full opacity (`render/draw.go:29-39`). An all-zero tint means opaque
white (`render/draw.go:42-46`).

These rules make transparent black impossible through a normal programmatic descriptor.
`Tint.A` and `Alpha` also store related opacity state twice.

Accepted action:

Use explicit defaults or presence flags. Do not give valid color values a second
meaning.

### Error returns add little value

`Theme.DrawWidget`, `DrawWidgetPart`, and `SetSkinPart` only fail for a nil receiver.
Normal callers already own a non-nil theme.

The UI ignores these errors (`ui/draw.go:34-46`). The gallery checks them repeatedly.

Accepted action:

Remove error returns that only guard nil receivers. Keep errors for operations with
real runtime failures.

### Renderer neutrality is overstated

The README says that `ui` is renderer-neutral (`README.md:28-30`). `ui` imports `render`
and owns a concrete `*render.Theme` (`ui/ui.go:12-27`).

Importing `rtgui/ui` therefore pulls raylib transitively. The UI cannot use another
renderer.

This coupling is intentional. Correct the README and do not add a renderer abstraction.

### UTF-8 storage does not mean Unicode rendering

The text buffer can store UTF-8. The font loader rasterizes printable ASCII and four
punctuation characters (`render/theme.go:192-200`).

This limit comes from `fontCodepoints`, not primarily from the Grenze font file. Raylib
does not rasterize a TTF glyph when that rune is absent from the requested set.

`Noto Sans` is a good general UI default. Separate Noto fonts cover CJK scripts and many
other writing systems. No practical single font covers all Unicode text.

Accept caller-defined rune sets and fallback fonts. A dynamic glyph cache is a later
option. Do not rasterize all Unicode glyphs into every size-specific atlas.

### The public API lacks release hygiene

- The current module path is `rtgui` (`go.mod:1`). This path works inside the local main
  module.
- Change the module path to the confirmed public path: `github.com/draxxris/rtgui`.
- `README.md:59` references the missing `examples/ui_sample` directory.
- The `pure-Go` title is inaccurate for consumers because raylib requires native build
  support.
- The C toolchain requirement is valid for Windows cross-builds and builds that import
  `render` or `ui`. It is not only a gallery requirement.
- The repository has no software license. The font license covers only the font files.
- Many exported declarations lack Go documentation.
- Many functions over eight lines lack the required doc comment.
- `go mod tidy -diff` reports changes.
- `github.com/vanng822/css` is a direct import but is marked indirect.

Move `github.com/vanng822/css` to the direct requirement block. The dependency has no
`go.mod`, so a tidy operation also changes the inherited module graph.

## Duplicate code assessment

The clone detector reports 23 clone groups at a 40-token threshold. Many groups are
harmless symmetry in tests or geometry.

The meaningful duplication is architectural. The accepted cleanup resolves it as
follows:

1. Remove all three hash implementations with numeric IDs.
2. Make `sim.Stage` use shared UI state transitions instead of copied interaction code.
3. Remove `assets.Registry` with the `assets` package.
4. Consolidate the regular and italic font cache lifecycle.
5. Consolidate callback registration and dispatch without a broad generic framework.
6. Remove exact hit-test and padding relay functions.
7. Make CSS and programmatic skin registration publish the same descriptor form.

Do not create a generic framework for every small duplicate. Remove the redundant owners
first.

## Recommended cleanup order

01. Change the module path and correct dependency metadata.
02. Remove `assets`.
03. Replace the text buffer with private in-place storage.
04. Replace drag-and-drop globals with an instance controller.
05. Add bounded draw recording and simplify render contracts.
06. Collapse widget, UI, and `sim` interaction state into one semantic path.
07. Simplify CSS tint, texture ownership, and registry publication.
08. Replace layout special cases with `SetPoint` and one traversal model.
09. Remove the remaining accepted fossils, but retain `layout.AnchorName`.
10. Correct API documentation and README claims.
11. Record the unresolved software-license decision without selecting a license.

## Candidates to retain

Not every small package or repeated branch is a defect.

- Keep the direct raylib boundary in `render`.
- Keep pure geometry types in `core`.
- Keep deterministic nine-patch geometry.
- Keep a headless render recorder, but make it explicit and bounded.
- Keep the skin registry fallback from a widget state to its normal state.
- Keep simple switch statements when they express a closed enum clearly.

The cleanup must remove owners and state. It must not hide the same complexity behind
new interfaces.

## Validation record

The following commands ran against commit `4cba79f`:

```text
mise run test                         PASS in the current worktree
go test -race ./...                   PASS in the current worktree
go test -shuffle=on -count=20 ./...  PASS in the current worktree
mise run vet                          PASS
mise run complexity                   PASS
gofmt -l                              PASS, no output
go build ./...                        PASS on linux/amd64
govulncheck ./...                     PASS, no vulnerabilities found
staticcheck ./...                     FAIL, two SA4006 reports
go mod tidy -diff                     FAIL, module files differ
clean archive: go test ./render       FAIL, missing Kenney images
```

The gallery ran for three frames under Xvfb in the current worktree. It produced a real
screenshot with the ignored local assets.

No automated pixel comparison exists. No test covers successful public CSS loading
without a display.

The audit did not inspect external consumer code because the repository has no
configured remote. The repository owner supplied the external contracts recorded in
this document.

## Final public-surface review

The implementation now uses the canonical module path
`github.com/draxxris/rtgui`. The public package set is `core`, `widgets`, `text`,
`transform`, `skin`, `render`, `layout`, `dragdrop`, `sim`, and `ui`. The removed
`assets` and `input` packages are not documented as supported packages.

The surviving public surface was reviewed with `go doc` for every package and an
AST-assisted declaration audit. Exported types, constants, errors, functions,
methods, and renderer-facing fields now have package-appropriate documentation.
Long production functions also have explanatory comments where their behavior is
not obvious. The review specifically covers the following retained contracts:

- `layout.AnchorName` remains the diagnostic spelling boundary for the `Anchor`
  enum.
- `dragdrop.Payload.ID` remains an application drag-source identity, not a
  widget or capture ID.
- `skin.Texture.ID` and raylib texture handles remain renderer resource IDs.
- `core.Color` is exact RGBA data; a zero value is transparent black.
- `render.DrawRecorder` is optional, frame-scoped, bounded, and defensive.
- CSS-created textures are owned by `render.Theme` and are released by
  `UnloadSkin` while the graphics context is active; programmatic textures are
  borrowed.
- `sim.Stage` borrows an existing `ui.UI` and shares its semantic callbacks.
- `layout.SetPoint` reports graph and constraint errors instead of silently
  overriding contradictory relations.

The accepted fossils were searched repository-wide and manually reviewed. No
source or active user documentation contains the removed `rtgui/input`,
`WidgetInfo.ID`, `HasCapture`, `StableID`, `TintImage`, `ApplyPadding`,
`EnforceMinSize`, `DebugBounds`, or `EstimateDrawCalls` contracts. The remaining
text matches are historical evidence in this audit and the execution plan, or
the valid resource identities listed above. `core.WidgetInfo.IsClipped` and
`ClipRect` were also removed; clipping remains an application-owned transform or
scissor concern rather than a renderer snapshot fossil.

The README and testing guide now describe raylib's direct/native-toolchain
requirement, the concrete `ui`/`render` relationship, the instance-owned drag
controller, UI-owned interaction, simulation callbacks, bounded recording, CSS
cleanup timing, exact colors, point relations, and the intentionally limited font
codepoint set. The missing `examples/ui_sample`, renderer-neutral claim,
`pure-Go` title, asset-package guidance, capture documentation, numeric-ID
guidance, stretch flags, and CPU tint-baking claims were removed.

The clean-checkout image limitation remains accepted and is not hidden: ignored
Kenney fixtures are available only in the current worktree, so an archive without
them cannot run the file-dependent CSS test or gallery skin. No ignored image or
`.gitignore` rule was changed. There is still no software license; selecting one
remains an unresolved release item for the repository owner.

The final validation commands and their exact outputs are recorded in the
execution record below after they run in the current worktree.

## Final validation record

- `git status --short` — PASS; only the expected cleanup, documentation, and
  previously untracked audit/benchmark/recorder files are present. The ignored
  `testdata/skins/kenney/` directory remains ignored and unchanged.
- `git diff --check` — PASS; no whitespace diagnostics.
- `gofmt -l $(find . -name '*.go' -type f)` — PASS; no output.
- `go mod tidy -diff` — PASS via `mise exec -- go`; no output.
- `go list ./...` — PASS; all 11 packages list under
  `github.com/draxxris/rtgui/...`.
- `DISPLAY= WAYLAND_DISPLAY= mise run test` — PASS; all packages passed.
- `go test -race ./...` — PASS via `mise exec -- go`; all packages passed with
  no race reports.
- `go test -shuffle=on -count=20 ./...` — PASS via `mise exec -- go`; all 20
  shuffled runs passed.
- `mise run vet` — PASS; no diagnostics.
- `staticcheck ./...` — the direct command could not find the mise-managed
  `go` binary; `mise exec -- staticcheck ./...` — PASS with no diagnostics.
- `mise run complexity` — PASS; no function exceeded complexity 15.
- `go build ./...` — PASS via `mise exec -- go`; no output.
- `govulncheck ./...` — PASS via the mise-managed tool; `No vulnerabilities
  found.`
- `go test -run '^$' -bench . -benchmem ./render ./layout ./transform` — PASS
  via `mise exec -- go`: `BenchmarkDrawWidgetPart` 60.50 ns/op, 0 B/op,
  0 allocs/op; `BenchmarkNinePatch` 12.42 ns/op, 0 B/op, 0 allocs/op;
  `BenchmarkArrangeWarmTree` 938.9 ns/op, 0 B/op, 0 allocs/op; and
  `BenchmarkMapping` 1.553 ns/op, 0 B/op, 0 allocs/op.
- `mise run build-linux` — PASS; `dist/widget-gallery-linux` built.
- `mise run build-windows` — PASS; `dist/widget-gallery.exe` cross-compiled
  with MinGW-w64.
- `xvfb-run --auto-servernum --server-args="-screen 0 1280x800x24" mise exec
  -- go run ./examples/widget_gallery -frames 3 -screenshot
  /tmp/rtgui-audit.png` — PASS; llvmpipe rendered three frames, reported
  `skin texture cleanup complete`, and wrote a 1280×780, 653092-byte PNG.
  Manual inspection found the textured/tinted panels and controls, nine-patch
  borders, text, slider/progress, layout child movement, state samples, and
  dropdown control visible; popup interaction remains covered by UI tests.
- CSS cleanup/no-recorder confirmation:
  `mise exec -- go test ./render -run
  'TestThemeDoesNotRecordWithoutRecorder|TestCSSUploadsUniqueSourceOnceAndPreservesDrawTint|TestCSSUploadFailureRollsBackCandidateAndPreservesOldLayer|TestSuccessfulCSSReloadUnloadsEachPriorTexture|TestUnloadAndClearSkinNeverUnloadBorrowedTextures'
  -count=1 -v` — PASS; all five tests passed.

The direct-toolchain note is environmental rather than a library failure: the
repository shell does not expose a bare Go binary, so Go-dependent commands were
rerun through `mise exec -- go` or the equivalent mise-managed tool.
