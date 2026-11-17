package services

import (
	"webtyp.com/appkit/backend"
	"webtyp.com/events/inprocess"
	"webtyp.com/mcp"
	"webtyp.com/rpc"
)

// Backend is what runs in the backend of the fixture app: no module.
//
// Example: built, err := backend.Build(config.App, services.Backend, services.Server, dsn, services.Schema)
var Backend = backend.Backend{
	Events: &inprocess.Broker{},
	API:    rpc.JSON{},
	Agents: mcp.Tools{Allowed: MCPTools()},
}
