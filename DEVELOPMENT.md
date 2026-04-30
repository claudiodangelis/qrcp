# Development

## Versioning

`qrcp` uses [semver](https://semver.org) for releases.

Version number is defined in `cmd/version.go`.

## Releases

We are using [goreleases](https://goreleaser.com/), [nfpm](https://nfpm.goreleaser.com/) and [Github Actions](https://github.com/features/actions) to build, package and release `qrcp`.

The relevant files are:

- .goreleases.yml
- .github/workflows/main.yml

The release action is triggered when a tag is pushed to the master branch.

## Web UI

The HTML templates served during send and receive operations live in the `web/` directory:

- `web/upload.html` — the upload page shown on `qrcp receive`
- `web/done.html` — the confirmation page shown after a successful upload

Both files are embedded into the binary at build time via the Go `embed` package (`web/web.go`). To modify the UI, edit those HTML files directly; no code changes are required.

The templates use Go's `html/template` syntax. Available variables:

| Template | Variable | Description |
|---|---|---|
| `upload.html` | `{{.Route}}` | The POST route for the upload form |
| `done.html` | `{{.File}}` | Comma-separated list of transferred file paths |

## Development workflow

1. Open a PR
2. Let someone review it
3. Squash commits and merge to master
4. When ready to release, add a tag
5. Wait for Github Action to process the release
