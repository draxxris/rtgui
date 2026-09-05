// Package sim provides a headless semantic adapter for RPC-controlled tests.
//
// A Stage borrows one existing ui.UI. Callers register widgets and callbacks
// with that UI before attaching the adapter. Simulated activation, focus, and
// typing use the same UI mutation helpers and callback ordering as physical
// input without inventing pointer coordinates or hover state.
//
// UI and Stage follow the library's single-owning-goroutine contract. They do
// not add mutexes or support concurrent physical and simulated operation.
package sim
