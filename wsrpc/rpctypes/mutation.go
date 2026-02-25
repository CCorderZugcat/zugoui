package rpctypes

// Handle assosciates a binding when sent from the server to the browswer
// Action associates a binding when sent from the browser to the server

// SetValueReq SetValue request
type SetValueReq struct {
	Handle int64  // binding ID
	Action string // action name
	Key    string // key name
	Value  any    // updated value
}

// InsertValueAtReq inserts a value inside or at the end of a slice.
// slices only
type InsertValueAtReq struct {
	Handle int64  // binding ID
	Action string // action name
	At     int    // position to insert before (if length, then append)
	Value  any    // new value
}

// RemoveValueAtReq removes a value from a slice. Any values after At are moved back.
// slices only
type RemoveValueAtReq struct {
	Handle int64  // binding ID
	Action string // action name
	At     int    // position to remove, if there are elements after, they move back
}

// SetValueAtReq sets a value at a given array or slice index.
// slices or arrays
type SetValueAtReq struct {
	Handle int64  // binding ID
	Action string // action name
	At     int    // position to replace
	Value  any    // replacing value
}

// SetValueForReq sets a key value.
// maps only
type SetValueForReq struct {
	Handle int64  // binding ID
	Action string // action name
	Key    string // key
	Value  any    // value
}

// RemoveValueForReq removes a key value.
// maps only
type RemoveValueForReq struct {
	Handle int64  // binding ID
	Action string // action name
	Key    string // key
}
