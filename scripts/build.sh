#!/usr/bin/env bash
set -ex

OWNER=ninjasphere
BIN_NAME=driver-go-led-controller
PROJECT_NAME=driver-go-led-controller


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
	git clone -b 'rpc2' git@github.com:ninjasphere/go-ninja.git $GOPATH/src/github.com/ninjasphere/go-ninja
fi

if [ ! -d $GOPATH/src/github.com/ninjasphere/github.com/ninjasphere/driver-go-gestic ]; then
	# Clone our internal gestic package
	git clone git@github.com:ninjasphere/driver-go-gestic.git $GOPATH/src/github.com/ninjasphere/driver-go-gestic
fi


# move the working path and build
cd .gopath/src/github.com/${OWNER}/${PROJECT_NAME}
go get -d -v ./...
go build -ldflags "-X main.GitCommit ${GIT_COMMIT}${GIT_DIRTY}" -o ${BIN_NAME}
mv ${BIN_NAME} ./bin
