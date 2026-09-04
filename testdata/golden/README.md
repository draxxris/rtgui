# Golden shots

Approved visual regression shots kept as a manual reference. Generate a fresh
gallery image with `go run ./examples/widget_gallery -frames 3 -screenshot
./shots/gallery.png` (or wrap the command with `xvfb-run`) and inspect it
against these files.

Each PNG is a deterministic scene per v1 widget x skin state plus no-skin fallback.
Fixed viewport 800x600, fixed atlas, no RNG, injected clock.

Later runs are eyeballed against these goldens; explain any difference before
accepting a replacement. There is no automated pixel-difference gate because
output can vary across drivers and raylib versions.
