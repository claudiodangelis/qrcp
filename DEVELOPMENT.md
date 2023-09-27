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
