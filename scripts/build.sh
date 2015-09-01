#!/usr/bin/env bash
set -ex

OWNER=ninjasphere
BIN_NAME=sphere-go-led-controller
PROJECT_NAME=sphere-go-led-controller


# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

GIT_COMMIT="$(git rev-parse HEAD)"
GIT_DIRTY="$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)"
VERSION="$(grep "const Version " version.go | sed -E 's/.*"(.+)"$/\1/' )"

# remove working build
# rm -rf .gopath
if [ ! -d ".gopath" ]; then
	mkdir -p .gopath/src/github.com/${OWNER}
	ln -sf ../../../.. .gopath/src/github.com/${OWNER}/${PROJECT_NAME}
fi

export GOPATH="$(pwd)/.gopath"

if [ ! -d $GOPATH/src/github.com/ninjasphere/go-ninja ]; then
	# Clone our internal commons package
	git clone git@github.com:ninjasphere/go-ninja.git $GOPATH/src/github.com/ninjasphere/go-ninja
fi

# move the working path and build
cd .gopath/src/github.com/${OWNER}/${PROJECT_NAME}
go get -d -v ./...
cd $GOPATH/src/github.com/golang/freetype && git checkout 5193f9f147f37ac3b321f80eb7798c9ca74be908

# building the master branch on ci
if [ "$BUILDBOX_BRANCH" = "master" ]; then
	go build -ldflags "-X main.BugsnagKey ${BUGSNAG_KEY}" -tags release -o ./bin/${BIN_NAME}
else
	go build -o ./bin/${BIN_NAME}
fi
