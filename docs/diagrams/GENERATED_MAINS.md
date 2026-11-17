# How the main files of an app are generated

Shows how `appkitc` writes the `main` files of an app and how a template that no longer matches the libraries is stopped before it is published. `webtyp dev` triggers it on start and when the models or `go.mod` change. Verified by [tests/fixtureapp_test.go](../../tests/fixtureapp_test.go) and [tests/maingen_test.go](../../tests/maingen_test.go).

## Before: the template was text inside `webtyp.com/server`
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

## Now: the generator is compiled against the libraries before it is published
```mermaid
flowchart TD
  A["webtyp dev<br/>(program with appkitc inside)"] --> B["server handler<br/>SetMainGenerator"]
  B --> C["appkitc.GenerateMain"]
  C --> D["writes web/server/main.go<br/>web/browser/main.go, migrate/main.go,<br/>config/services"]
  D --> E{"did any file change?"}
  E -- "yes" --> F["go get the libraries<br/>the templates call"]
  E -- "no" --> G["go build"]
  F --> G
```
1. `webtyp dev` starts the server handler; the program connects `appkitc` to it before the first generation. Without one connected, the handler fails with `server.ErrNoMainGenerator`.
2. `GenerateMain` writes the files of the app (never a file without the generated header).
3. If any file changed it runs `go get <library>@latest` for each library the templates call, then `go build` succeeds.
4. The app's `go.mod` never lists `appkitc`.

## How a template that no longer matches the libraries is stopped
```mermaid
flowchart TD
  P1["appkit changes and is published"] --> P2["the cascade republishes appkitc"]
  P2 --> P3{"appkitc test:<br/>do the generated mains compile<br/>against the new appkit?"}
  P3 -- "no" --> P4["fails WHEN PUBLISHING<br/>not in anyone's dev"]
  P3 -- "yes" --> P5["new appkitc version published"]
  P5 --> P6["the cascade republishes app<br/>and gopush installs webtyp"]
```
1. `appkit` changes and is published.
2. The cascade republishes `appkitc`; its fixture test generates the `main` files and compiles them against the new `appkit`.
3. If they do not compile, publishing fails there, never in somebody's `webtyp dev`.
4. If they compile, `app` is republished and the installed `webtyp` carries the new templates; an older installed `webtyp` still writes older templates until it is reinstalled.
