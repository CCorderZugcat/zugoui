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
	*Server
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
	// unsupported
}

// RemoveValueAt stub for function [wsrpc.Server.RemoveValueAt]
func (o Observer) RemoveValueAt(at int) {
	// unsupported
}

// SetValueAt stub for function [wsrpc.Server.SetValueAt]
func (o Observer) SetValueAt(at int, value any) {
	// unsupported
}

// SetValueFor stub for function [wsrpc.Server.SetValueFor]
func (o Observer) SetValueFor(key string, value any) {
	// unsupported
}

// RemoveValueFor stub for function [wsrpc.Server.RemoveValueFor]
func (o Observer) RemoveValueFor(key string) {
	// unsupported
}
