// Command appkitc writes the generated main files of the webtyp appkit app in the current folder.
//
// Example: go tool appkitc -port=8080 -public-dir=web/public
package main

import (
	"flag"
	"fmt"
	"os"

	"webtyp.com/appkitc"
	"webtyp.com/server"
)

func main() {
	port := flag.String(server.GeneratorFlagPort, "", "port the generated server listens on")
	publicDir := flag.String(server.GeneratorFlagPublicDir, "", "folder the generated server serves, relative to the app")
	plainHTTP := flag.Bool(server.GeneratorFlagPlainHTTP, false, "serve without TLS")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg := server.MainConfig{Port: *port, PublicDir: *publicDir, PlainHTTP: *plainHTTP}
	path, err := appkitc.GenerateMain(root, server.ReadModulePath(root), cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
}
