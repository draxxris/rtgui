# Code Standards

Keep cyclomatic complexity under 15. Use the following tool to check, before labeling
your code as "complete": `mise run complexity`

Functions over 8 lines long must have a docstring.

Files should not exceed 4000 lines in length. This is not a hard requirement, just a
strong recommendation.

Do not mention plan phases in documention or source code -- it is too ambiguous.

# Go Memory Allocation

Minimize heap allocations and GC pressure in hot paths, especially ECS ticks, per-entity
updates, and networking. Prefer value types, packed component storage,
preallocated/reused slices and buffers, ring buffers, and direct packet encode/decode
into reusable memory. Avoid per-entity/per-packet heap objects, unnecessary
pointers/interfaces, repeated slice growth, and per-tick temporary allocations.

Aim for zero allocations in steady-state simulation and packet processing where
practical. Use `sync.Pool` selectively for high-frequency temporary objects and packet
buffers; it can reduce allocation pressure but adds ownership/lifetime complexity,
pooled objects may be discarded by the runtime, and retaining oversized buffers can
waste memory. Prefer naturally allocation-free designs first, and use escape analysis
when investigating unexpected heap allocations. Benchmarking and allocation profiling
are performed separately as part of periodic performance testing.
