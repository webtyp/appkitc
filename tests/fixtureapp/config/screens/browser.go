package screens

import (
	"webtyp.com/appkit/browser"
	"webtyp.com/model"
	"webtyp.com/rpc"
	"webtyp.com/unixid"
)

// Browser is what runs in the browser of the fixture app: no module screens.
//
// Example: browser.Run(config.App, screens.Browser, layoutapp.New(devtools.Menu()))
var Browser = browser.Browser{
	Caller: rpc.NewCaller(""),
	NewIDs: func() (model.IDGenerator, error) { return unixid.NewUnixID() },
}
