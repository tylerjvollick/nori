package dtos

import "encoding/json"

// Nullable distinguishes between a JSON field that is absent, explicitly
// null, and set to a value. Use it in update (PATCH-style) request DTOs
// where "absent" means "leave unchanged" and "null" means "clear".
//
//	Set == false              → field absent: no change
//	Set == true, Valid == false → field was null: clear the value
//	Set == true, Valid == true  → field was provided: set Value
type Nullable[T any] struct {
	Set   bool
	Valid bool
	Value T
}

// UnmarshalJSON implements json.Unmarshaler. It is only invoked when the
// field is present in the JSON payload, which is how absence is detected.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.Set = true
	if string(data) == "null" {
		n.Valid = false
		var zero T
		n.Value = zero
		return nil
	}
	if err := json.Unmarshal(data, &n.Value); err != nil {
		return err
	}
	n.Valid = true
	return nil
}

// MarshalJSON implements json.Marshaler so the type round-trips cleanly.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if !n.Set || !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Value)
}

// Ptr returns a pointer to the value when set and non-null, or nil when the
// field was explicitly null. Only meaningful when Set is true.
func (n Nullable[T]) Ptr() *T {
	if !n.Valid {
		return nil
	}
	v := n.Value
	return &v
}
