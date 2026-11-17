//go:build !wasm

package services

import (
	"webtyp.com/appkit/backend"
	"webtyp.com/model"
	"webtyp.com/sqlite"
	"webtyp.com/unixid"
)

// Server is what only the native server decides: the database driver and how ids are made.
//
// Example: built, err := backend.Build(config.App, services.Backend, services.Server, dsn, services.Schema)
var Server = backend.Server{
	Database: sqlite.Open,
	NewIDs:   func() (model.IDGenerator, error) { return unixid.NewUnixID() },
}
