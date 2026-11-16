# Heartless

A native desktop API client for HTTP, built in Go. It uses the
[MyGo](https://github.com/egoist/mygo) framework, so there is no Electron, no
bundled browser, and nothing to install alongside it but the system's own
WebKit and GTK.

Heartless is a rebuild of the [Yaak](https://yaak.app) API client using Go.
Requests, collections, environments and responses live in a local SQLite
database, and the window draws itself.

## Status

Early development. The HTTP core, storage layer and request editor work end to
end; the remaining phases in [`docs/PLAN.md`](docs/PLAN.md) are in progress. See
[`docs/PROGRESS.md`](docs/PROGRESS.md) for what is done and what is not.

## Features

**Requests**

- HTTP methods (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS) with a URL field
  that previews how `{{variables}}` resolve.
- Query parameters and headers as editable rows, each row individually
  enabled or disabled.
- Body types: URL-encoded, multipart, GraphQL, JSON, plain text, binary. JSON
  bodies written as JavaScript objects are converted to strict JSON, and the
  preview highlights the result.
- Authentication: bearer token, basic, and API key. The menu lists the other
  schemes (AWS, Digest, JWT, OAuth 1.0/2.0, NTLM) for when they land, but only
  the first three are applied to the request today.

**Post-response scripting**

Each request can carry a JavaScript snippet that runs automatically when the
response arrives:

```js
env.set("token", res.json().token);
test("succeeds", () => {
  if (res.status !== 200) throw new Error("got " + res.status);
});
console.log("took", req.method, "to", res.url);
```

- `req` — `id`, `method`, `url`, `body`, `bodyType`, `params`, `headers`,
  `authType`, `workspaceId`
- `res` — `status`, `statusText`, `headers`, `body`, `json()`, `size`,
  `contentType`, `url`, `error`
- `env.get(name)`, `env.set(name, value)`, `env.has(name)`, `env.all()`
- `test(name, fn)` for assertions, and `console.log/info/debug/warn/error`

Scripts run in an isolated [goja](https://github.com/dop251/goja) runtime with
no network, filesystem or process access, interrupted after two seconds. Runs
happen on their own goroutine, so a slow script never blocks the interface.
Variables a script changes are shown as `old → new` and written only when you
save them.

**Environments and variables**

- Workspaces, environments and variables stored in SQLite.
- `{{variable}}` substitution with environment values taking precedence over
  workspace ones.
- The editor highlights a placeholder that resolves and flags one that does not.
- One-click switching of the active environment from the title bar.

**Collections**

- Collections, nested folders and saved requests in a sidebar tree.
- Rename and delete requests, and import Postman and Insomnia collections.
- Double-click behaviour is configurable in settings.

**History**

Every send is recorded with its response, so a request's last result is still
there when you reopen it. Entries are pruned by a configurable retention limit.

**Interface**

- Light and dark themes with adjustable accent, background and foreground
  colours, plus an interface font size.
- A custom title bar with the workspace and environment breadcrumb.
- Keyboard-friendly request editor with per-request drafts.

## Install

### Linux

```sh
curl -fsSL https://raw.githubusercontent.com/heisdezz/heartless/main/scripts/linux-install.sh | sh
```

The script picks the newest release and installs it the way your distribution
supports: the `.deb` through `apt` on Debian and Ubuntu, so dependencies are
resolved, or the `.tar.gz` into `~/.local` everywhere else, which needs no root.
Pass `--dry-run` to see what it would do.

Heartless needs WebKitGTK and GTK 3 at run time. The `.deb` declares them; for
other distributions:

```sh
# Arch
sudo pacman -S webkit2gtk-4.1

# Fedora
sudo dnf install webkit2gtk4.1

# Debian / Ubuntu
sudo apt-get install -y libwebkit2gtk-4.1-0
```

Only `linux/amd64` builds are published at the moment.

### macOS and Windows

Builds are produced by CI for macOS (universal) and Windows, but there is no
installer script for them yet. Until there is, build from source.

## Build from source

You need Go 1.27 or newer, and the platform's WebView development packages on
Linux:

```sh
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev libgl1-mesa-dev
```

Then:

```sh
git clone https://github.com/heisdezz/heartless
cd heartless

go tool mygo dev      # run with live reload
go tool mygo build    # production build into build/
```

`go tool mygo build -platform linux/amd64` also writes a `.deb`, an `.rpm`, a
PKGBUILD and a `.tar.gz` bundled with an `install.sh`.

Database location:

- Linux — `~/.config/heartless/heartless.db`
- macOS — `~/Library/Application Support/heartless/heartless.db`
- Windows — `%AppData%\heartless\heartless.db`

## Development

```sh
go test ./...        # all tests
go vet ./...         # vet
gofmt -l .           # files needing formatting
```

Architecture, conventions and the phased roadmap are in
[`docs/PLAN.md`](docs/PLAN.md). Feature-by-feature reference notes, carried over
from Yaak, are in [`docs/features/`](docs/features/). The UI layout is described
in [`docs/ui/`](docs/ui/).

### Project layout

```
main.go                  window setup and app bootstrap
internal/app/            service container and dependency wiring
internal/domain/         entities: request, response, collection, environment,
                         variable, history, workspace, variable resolution
internal/services/       request execution, importers, history, auth
internal/storage/sqlite/ repositories and migrations
internal/transport/http/ HTTP client
internal/screens/        every screen, including the scripting runtime
internal/config/         settings and theme configuration
internal/events/         in-process event bus
internal/logging/        structured logging with redaction
migrations/              initial schema
scripts/                 release tooling and the Linux installer
```

## Commit messages

Releases are versioned from commit messages, so they follow
[Conventional Commits](https://www.conventionalcommits.org/). See
[`commit.md`](commit.md) for the types and how CI derives the next version.

## License

No licence file is present yet. Until one is added, the code is all rights
reserved, so contact the author before reusing it.
