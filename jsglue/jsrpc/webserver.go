//go:build js

package jsrpc

import (
	"net/rpc"

	"github.com/CCorderZugcat/zugoui/wsrpc/rpctypes"
)

// Server client stub for [wsrpc.Server]
type Server struct {
	*rpc.Client
}

// Observer implements the Observer interface to the server with the Action
type Observer struct {
	Server
	Action string
}

// Action client side stub for function [wsrpc.Server.Action]
func (s Server) Action(action string) {
	go s.Call(
		"Server.Action",
		&rpctypes.ActionReq{
			Action: action,
		},
		nil,
	)
}

// SetValue client stub for function [wsrpc.Server.SetValue]
func (o Observer) SetValue(key string, value any) {
	go o.Call(
		"Server.SetValue",
		&rpctypes.SetValueReq{
			Action: o.Action,
			Key:    key,
			Value:  value,
		},
		nil,
	)
}

// InsertValueAt stub for function [wsrpc.Server.InsertValueAt]
func (o Observer) InsertValueAt(at int, value any) {
	go o.Call(
		"Server.InsertValueAt",
		&rpctypes.InsertValueAtReq{
			Action: o.Action,
			At:     at,
			Value:  value,
		},
		nil,
	)
}

// RemoveValueAt stub for function [wsrpc.Server.RemoveValueAt]
func (o Observer) RemoveValueAt(at int) {
	go o.Call(
		"Server.RemoveValueAt",
		&rpctypes.RemoveValueAtReq{
			Action: o.Action,
			At:     at,
		},
		nil,
	)
}

// SetValueAt stub for function [wsrpc.Server.SetValueAt]
func (o Observer) SetValueAt(at int, value any) {
	go o.Call(
		"Server.SetValueAt",
		&rpctypes.SetValueAtReq{
			Action: o.Action,
			At:     at,
			Value:  value,
		},
		nil,
	)
}

// SetValueFor stub for function [wsrpc.Server.SetValueFor]
func (o Observer) SetValueFor(key string, value any) {
	go o.Call(
		"Server.SetValueFor",
		&rpctypes.SetValueForReq{
			Action: o.Action,
			Key:    key,
			Value:  value,
		},
		nil,
	)
}

// RemoveValueFor stub for function [wsrpc.Server.RemoveValueFor]
func (o Observer) RemoveValueFor(key string) {
	go o.Call(
		"Server.RemoveValueFor",
		&rpctypes.RemoveValueForReq{
			Action: o.Action,
			Key:    key,
		},
		nil,
	)
}
