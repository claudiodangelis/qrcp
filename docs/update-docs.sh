#!/bin/bash
if ! [[ $(git rev-parse --show-toplevel 2>/dev/null) = "$PWD" ]]; then
    echo "error: script should be run from the root of the repository"
    exit 1
fi
cp README.md docs/index.md
find docs -name "*.md" | while read page
do
    sed -i 's/docs\/img/img/g' $page
done
