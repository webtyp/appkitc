package config

import (
	"webtyp.com/appkit"
	"webtyp.com/country"
	"webtyp.com/fmt"
	"webtyp.com/model"
)

// App is the smallest app the generated mains compile against.
//
// Example: browser.Run(config.App, screens.Browser, layoutapp.New(devtools.Menu()))
var App = appkit.App{
	Name:      "Fixture",
	TenantID:  "fixture",
	Timezone:  "America/Santiago",
	Countries: country.Catalog{Default: "CL"},
	SignIn:    appkit.SignInByTrustedIP,
	Modules:   []model.ModuleDefinition{},
	Log:       func(a ...any) { fmt.Println(a...) },
}
