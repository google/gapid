#!/bin/bash

DO_DIR="`dirname \"${BASH_SOURCE[0]}\"`"
DO_DIR="`( cd \"$DO_DIR\" && pwd )`"

if [ ! -d "$DO_DIR/../../../../src/github.com/google/gapid" ]; then
  echo "It looks like the directory structure of"
  echo "  $DO_DIR"
  echo "doesn't match what is expected."
  echo
  echo "Did you 'git clone' the project instead of using 'go get'?"
  echo "Please follow the building directions found here:"
  echo "  https://github.com/google/gapid/blob/master/BUILDING.md"
  exit 1
fi

export GOPATH="$DO_DIR/third_party:`( cd \"$DO_DIR/../../../../\" && pwd )`"
cd ${DO_DIR} && go run ./cmd/do/*.go "$@"
