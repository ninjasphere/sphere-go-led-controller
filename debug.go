// +build !release

package main

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/juju/loggo"
	"github.com/ninjasphere/go-ninja/logger"
)

func init() {
	logger.GetLogger("").SetLogLevel(loggo.DEBUG)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       "ddda18251cc5146dc9e3d2ac89e14abb",
		ReleaseStage: "development",
	})
}
