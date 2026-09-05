// Package dragdrop provides instance-owned drag sessions and drop targets.
//
// It stores application payloads as opaque Go values and does not depend on
// widget, game, or renderer types.
package dragdrop

// Payload carries an external string identity alongside its kind and data.
// The ID names the drag source; it is empty when the source is anonymous.
type Payload struct {
	// Kind identifies the application payload category.
	Kind string
	// Data is the application-owned payload value.
	Data any
	// ID identifies the drag source when the source is named.
	ID string
}

// NewPayload returns a Payload with the given source ID, kind, and data.
func NewPayload(id string, kind string, data any) Payload {
	return Payload{ID: id, Kind: kind, Data: data}
}
