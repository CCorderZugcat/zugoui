package rpctypes

// DispatchEventReq DispatchEvent request
type DispatchEventReq struct {
	Type   string // event type
	Detail any    // event data
}

// NewValueBindingReq NewValueBinding request
type NewValueBindingReq struct {
	Action     string   // action name
	ElementIDs []string // element IDs
	Property   string   // property of the element (special case for "value")
	Model      any      // initial value and also establishes the type
}

// NewValueBindingRes NewValueBinding result
type NewValueBindingRes struct {
	Handle int64 // binding handle
}

// NewClickBindingReq NewClickBinding request
type NewClickBindingReq struct {
	ElementID string // element ID
	Action    string // action name
}

// NewClickBindingRes NewClickBinding result
type NewClickBindingRes struct {
	Handle int64 // binding handle
}

// UnbindReq Unbind request
type UnbindReq struct {
	Handle int64 // binding handle
}
