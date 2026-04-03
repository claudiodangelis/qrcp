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

## Development workflow

1. Open a PR
2. Let someone review it
3. Squash commits and merge to master
4. When ready to release, add a tag
5. Wait for Github Action to process the release

## Manual testing checklist

Before merging changes, verify the following scenarios work correctly. Build the binary first with `go build -o qrcp .`.

### Send (local to mobile)

- Basic send: `./qrcp README.md` — scan the QR code on a mobile browser and confirm the file downloads correctly
- Custom port: `./qrcp -p 9090 README.md` — verify the QR code URL uses port 9090 and the download works
- Large file: send a file >100MB and confirm it transfers completely without errors or corruption

### Receive (mobile to local)

- Basic receive: `./qrcp r` — scan the QR code on a mobile browser, upload a file, and confirm it appears in the current directory
- Large file: upload a file >100MB from mobile and confirm it is received completely

### Cross-browser compatibility

Test the QR code URL directly in each of the following mobile browsers and confirm send/receive both work:

- Safari (iOS)
- Chrome (Android)
- Firefox (Android)
- Samsung Internet
