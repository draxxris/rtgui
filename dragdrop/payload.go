package dragdrop

// Opaque Go drag payload — no game types.

// Payload carries an external string identity alongside its kind and data.
// The ID names the drag source; it is empty when the source is anonymous.
type Payload struct {
	Kind string
	Data any
	ID   string
}

// NewPayload returns a Payload with the given source ID, kind, and data.
func NewPayload(id string, kind string, data any) Payload {
	return Payload{ID: id, Kind: kind, Data: data}
}
