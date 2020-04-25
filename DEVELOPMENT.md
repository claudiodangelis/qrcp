# Development

## Versioning

`qrcp` uses [semver](https://semver.org).

Version number is defined in `cmd/version.go`.

## Releases

`qrcp` uses [goreleases](https://goreleaser.com/) to build the binaries for Linux, Mac and Windows and push them to Github.


```sh
# Define tag
QRCP_TAG=0.5.1
# Add tag
git tag -a $QRCP_TAG -m "$QRCP_TAG Release"
# Push branch
git push origin $QRCP_TAG
# Run the actual release script
goreleaser
```
