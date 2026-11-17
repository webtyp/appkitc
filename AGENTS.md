# AGENTS.md — `webtyp/appkitc`

<!-- webtyp:base — copied from devskills skill agents-md; do not edit here -->
## Ecosystem rules — read before any change

`docs/PLAN.md` is the spec: no design judgment; anything unclear is a question in the PR, not a guess.

- Public API: typed (no `any`/`interface{}`), explicit, sequential, minimal; no generics (`[T any]`) or `reflect` without benchmarked allocation/performance justification and owner approval; never export a symbol only a test uses.
- One intent, one path: the old path is deleted in the same change. No silent fallback, `TODO`, commented code or stub.
- Fail at compile time, else a loud diagnostic — never silently. Zero values deny.
- A defect in a dependency is reported, never wrapped or patched here.
- No string literals in logic: named constants. The same logic never lives in two repos: import the package the plan names; if you find an existing one the plan does not name that already does what you are about to write, say so in the PR instead of writing it.
- `cmd/*/main.go`: parse flags, inject, print/exit only. stdout = data, stderr = logs, exit codes are the contract.
- WASM code: `webtyp/fmt`, `webtyp/time`, `webtyp/json` instead of stdlib (imported without alias as `fmt`, `time`, `json` — never `tinytime`); no `map`; `dom.Element` embedded by value.
- `css.go`, `svg.go`, `js.go`, `html.go` carry `//go:build !wasm`; no `ssr.go`, no `front.go`.
- Build tags: `wasm`/`!wasm` only when the API does not exist on that platform; `browser`/`!browser` when an app's file belongs to another binary; a library carries no destination tag (the linker drops what a binary never calls). One main per folder, no destination tag: `web/browser/main.go` → `web/public/browser.wasm`, `web/server/main.go` → `web/server/server`, `web/edge/main.go` → `web/edge/edge.wasm`.
- Code and keys in English; UI text is `lang.Text`, translations live in `lang.json`. Comments and docs: English in `webtyp/*`; in `veltylabs/*` **every README, `docs/*` file, diagram and code comment is in Spanish**, written for a reader with little technical background, and each exported symbol's comment shows a short usage example.
- A translation key is reused, never duplicated as a near-identical phrase (`"not a valid integer"` + `"not a valid decimal number"` repeats `"not a valid"` three ways instead of composing it once) — every library ships to WASM/TinyGo, so each unreused phrase is dictionary rows AND binary bytes, in every consumer, forever. A token that doesn't need translating (a kind/type name like `int`) is passed as data (a variable), never as a string literal — a literal becomes a dictionary key whether or not it needed one.
- Tests cover behaviour and domain rules, never the look (CSS, colours, markup order, class strings). In `tests/` (external package), real stack with fakes only at the edge; run `gotest`.
- Docs (`README.md`, `docs/*`) updated with the API; permanent docs never link `docs/PLAN.md`.
- Before the PR: `git diff --stat main` lists every file of the plan's stages, and nothing the plan did not order is deleted.
<!-- /webtyp:base -->

## This repo

`appkitc` is a **development tool**: it writes the `main` files of a webtyp appkit app (`web/server`, `web/browser`, `migrate`, `config/services`). It never runs in production and no app imports it: each app pins it with a `tool` line in its `go.mod` and `webtyp dev` runs `go tool appkitc`.

- **Stdlib is legitimate here.** This repo runs on the developer's machine, never in the browser or in a server binary: keep `fmt`, `os`, `text/template`, `go/ast` as they are. Do NOT "fix" them to `webtyp/fmt`. (The files under `tests/fixtureapp/` are the exception: they are an app, so they use `webtyp/fmt`.)
- **Dependency direction:** `appkitc` imports `webtyp.com/server` (project detection, `MainConfig`, flag names) and the libraries the templates call; neither `server` nor `appkit` may ever import `appkitc`.
- **`tests/fixtureapp/` is a compiled fixture.** Its hand-written files (`config/app.go`, `config/services/{backend,mcp,server}.go`, `config/screens/browser.go`) are edited by hand; its generated files (`web/server/main.go`, `web/browser/main.go`, `migrate/main.go`, `config/services/{migrations,removed}.go`) are goldens: never edit them by hand, regenerate with `APPKITC_UPDATE_FIXTURE=1 gotest`.
- **Templates change only together with the library API they call.** A template that does not compile against the libraries in `go.mod` must fail `gotest` here, never in an app.
- Never write `go.mod`/`go.sum` lines by hand: only `go get` and `go mod tidy`.
