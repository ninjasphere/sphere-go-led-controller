# Ninja Sphere - Chromecast Driver


[![Build status](https://badge.buildkite.com/98d7e0d70ed84df9d1449426e30e09769f5b843f92456f2429.svg)](https://buildkite.com/ninja-blocks-inc/sphere-go-led-controller)
[![godoc](http://img.shields.io/badge/godoc-Reference-blue.svg)](https://godoc.org/github.com/ninjasphere/sphere-go-led-controller)
[![MIT License](https://img.shields.io/badge/license-MIT-yellow.svg)](LICENSE)
[![Ninja Sphere](https://img.shields.io/badge/built%20by-ninja%20blocks-lightgrey.svg)](http://ninjablocks.com)
[![Ninja Sphere](https://img.shields.io/badge/works%20with-ninja%20sphere-8f72e3.svg)](http://ninjablocks.com)

---


### Introduction
This application is part of Ninja Sphere, controlling the LED display and gesture system on the Spheramid.

It communicates with HomeCloud (using the ThingModel) in order to find the devices to control, enabling panes as needed.

### Requirements

* Go 1.3

### Dependencies

https://github.com/ninjasphere/gestic-tools/

### Building

Due to the native components of the golang gestic sdk, it is not possible to cross-compile. In order to build a binary that works correctly on the Spheramid, you will need to install golang on the target machine (or another linux/arm machine).

### Running

`go build && DEBUG=* ./sphere-go-led-controller`

You will likely want to first stop the built-in binary using `stop led-controller`

### Options

* `--enableControl` - Starts in control mode immediately.
* `--led.forceAllPanes` - Always enable all panes (including test ones)
* `--mqtt.host=HOST` - Override default mqtt host
* `--mqtt.port=PORT` - Override default mqtt host

Other options are available in `/opt/ninjablocks/config/default` (all options can be overridden by cli args or env vars).

### More Information

More information can be found on the [project site](http://github.com/ninjasphere/sphere-go-led-controller) or by visiting the Ninja Blocks [forums](https://discuss.ninjablocks.com).

### Contributing Changes

To contribute code changes to the project, please clone the repository and submit a pull-request ([What does that mean?](https://help.github.com/articles/using-pull-requests/)).

### License
This project is licensed under the MIT license, a copy of which can be found in the [LICENSE](LICENSE) file.

### Copyright
This work is Copyright (c) 2014-2015 - Ninja Blocks Inc.
