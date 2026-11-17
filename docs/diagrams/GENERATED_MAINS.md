# How the main files of an app are generated

Shows how `appkitc` writes the `main` files of an app and why its templates cannot be older than the libraries: the app's `go.mod` pins the version. `webtyp dev` triggers it on start and when the models or `go.mod` change. Verified by [tests/fixtureapp_test.go](../../tests/fixtureapp_test.go) and [tests/maingen_test.go](../../tests/maingen_test.go).

## Before: the template travelled inside the `webtyp` binary
```mermaid
flowchart TD
  A["webtyp dev<br/>installed binary: server v0.2.127"] --> B["text template<br/>INSIDE the binary"]
  B --> C["writes web/browser/main.go<br/>and migrate/main.go"]
  D["app go.mod<br/>appkit v0.2.38, layout v0.3.130"] --> E["go build"]
  C --> E
  E --> F["ERROR: undefined layoutapp.Browser<br/>old template, new API"]
```
1. `webtyp dev` is an installed binary; its template is the one of the version it was built with.
2. It rewrote the `main` files even when the app's `go.mod` used newer libraries.
3. `go build` failed: the old template called an API that no longer exists.

## Now: each `webtyp dev` runs the version the app pinned
```mermaid
flowchart TD
  A["webtyp dev<br/>no templates"] --> B["reads the app go.mod<br/>tool appkitc v0.1.3"]
  B --> C["go tool appkitc<br/>runs THAT version"]
  C --> D["writes web/browser/main.go<br/>migrate/main.go, config/services"]
  D --> E["go build succeeds"]
```
1. `webtyp dev` only reads the app's `go.mod` (`tool webtyp.com/appkitc/cmd/appkitc`) and runs `go tool appkitc`; if the line is missing it adds it first.
2. `go tool` builds `appkitc` the first time (a few seconds, then cached) and runs that exact version.
3. It writes `web/server/main.go`, `web/browser/main.go`, `migrate/main.go` and `config/services`; if any file changed it runs `go mod tidy`, then `go build` succeeds.
4. Error branches: no `go.mod` or the tool cannot be added → `server.ErrGeneratorNotInstalled`; the generator stops → `server.ErrGeneratorFailed`; each says what to run.

## How it stays in step with the libraries
```mermaid
flowchart TD
  P1["appkit changes and is published"] --> P2["the cascade republishes appkitc"]
  P2 --> P3{"appkitc test:<br/>do the generated mains compile<br/>against the new appkit?"}
  P3 -- "no" --> P4["fails WHEN PUBLISHING<br/>not in anyone's dev"]
  P3 -- "yes" --> P5["new appkitc version published"]
  P5 --> P6["in the app: go get -u<br/>appkit and appkitc rise together"]
  P6 --> P7["webtyp dev regenerates with the new template"]
```
1. `appkit` changes and is published.
2. The cascade republishes `appkitc`; its fixture test generates the `main` files and compiles them against the new `appkit`.
3. If they do not compile, publishing fails there, never in somebody's `webtyp dev`.
4. If they compile, the app runs `go get -u` and `appkit` and `appkitc` rise together; if only `appkit` rose, «Update libraries» runs `go get webtyp.com/appkitc@latest`.
