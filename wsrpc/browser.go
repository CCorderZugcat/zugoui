package wsrpc

import (
	"errors"
	"fmt"
	"net/rpc"

	"github.com/CCorderZugcat/zugoui/jstypes"
	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/wsrpc/rpctypes"
)

var ErrBadType = errors.New("bad type")

// rpc types for calling the browser side

// Browser client side stub for [jsrpc.Browser]
type Browser struct {
	*rpc.Client
}

// Observer implements the Observer interface to the browswer with the binding handle
type Observer struct {
	Browser
	Handle int64
}

var _ observable.Observer = Observer{}

// DispatchEvent client stub function for [jsrpc.Browser.DispatchEvent]
func (b Browser) DispatchEvent(name string, detail any) error {
	detail, ok := jstypes.ValueOf(detail)
	if !ok {
		return fmt.Errorf(
			"%w: could not convert %T to js friendly type",
			ErrBadType,
			detail,
		)
	}

	return b.Call(
		"Browser.DispatchEvent",
		&rpctypes.DispatchEventReq{
			Type:   name,
			Detail: detail,
		},
		nil,
	)
}

// NewValueBinding client stub function for [jsrpc.Browser.NewValueBinding]
func (b Browser) NewValueBinding(action, formID string, elementIDs []string, model any) (int64, error) {
	resp := &rpctypes.NewValueBindingRes{}
	if err := b.Call(
		"Browser.NewValueBinding",
		&rpctypes.NewValueBindingReq{
			Action:     action,
			FormID:     formID,
			ElementIDs: elementIDs,
			Model:      model,
		},
		resp,
	); err != nil {
		return -1, err
	}

	return resp.Handle, nil
}

// NewClickBinding client stub functionfor [jsrpc.Browser.NewClickBinding]
func (b Browser) NewClickBinding(elementID, action string) (int64, error) {
	resp := &rpctypes.NewClickBindingRes{}
	if err := b.Call(
		"Browser.NewClickBinding",
		&rpctypes.NewClickBindingReq{
			ElementID: elementID,
			Action:    action,
		},
		resp,
	); err != nil {
		return -1, err
	}

	return resp.Handle, nil
}

// Unbind client stub function for [jsrpc.Browser.Unbind]
func (b Browser) Unbind(handle int64) error {
	return b.Call(
		"Browser.Unbind",
		&rpctypes.UnbindReq{
			Handle: handle,
		},
		nil,
	)
}

// SetValue stub for function [jsrpc.Browser.SetValue]
func (o Observer) SetValue(key string, value any) {
	o.Browser.Call(
		"Browser.SetValue",
		&rpctypes.SetValueReq{
			Handle: o.Handle,
			Key:    key,
			Value:  value,
		},
		nil,
	)
}

// InsertValueAt stub for function [jsrpc.Browser.InsertValueAt]
func (o Observer) InsertValueAt(at int, value any) {
	o.Browser.Call(
		"Browser.InsertValueAt",
		&rpctypes.SetValueAtReq{
			Handle: o.Handle,
			At:     at,
			Value:  value,
		},
		nil,
	)
}

// RemoveValueAt stub for function [jsrpc.Browser.RemoveValueAt]
func (o Observer) RemoveValueAt(at int) {
	o.Browser.Call(
		"Browser.RemoveValueAt",
		&rpctypes.RemoveValueAtReq{
			Handle: o.Handle,
			At:     at,
		},
		nil,
	)
}

// SetValueAt stub for function [jsrpc.Browser.SetValueAt]
func (o Observer) SetValueAt(at int, value any) {
	o.Browser.Call(
		"Browser.SetValueAt",
		&rpctypes.SetValueAtReq{
			Handle: o.Handle,
			At:     at,
			Value:  value,
		},
		nil,
	)
}

// SetValueForm stub for function [jsrpc.Browser.SetValueFor]
func (o Observer) SetValueFor(key string, value any) {
	o.Browser.Call(
		"Browser.SetValueFor",
		&rpctypes.SetValueForReq{
			Handle: o.Handle,
			Key:    key,
			Value:  value,
		},
		nil,
	)
}

// RemoveValueFor stub for function [jsrpc.Browser.RemoveValueFor]
func (o Observer) RemoveValueFor(key string) {
	o.Browser.Call(
		"Browser.RemoveValueFor",
		&rpctypes.RemoveValueForReq{
			Handle: o.Handle,
			Key:    key,
		},
		nil,
	)
}
