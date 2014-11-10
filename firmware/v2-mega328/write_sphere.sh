#!/bin/sh

# callers can set SKIP_SERVICE_CONTROL to true to skip service control, but by default, we do it
# if true, caller has already arranged for services to be stopped

SKIP_SERVICE_CONTROL=${SKIP_SERVICE_CONTROL:-false}

$SKIP_SERVICE_CONTROL || stop sphere-leds || true
$SKIP_SERVICE_CONTROL || stop devkit-status-led || true

echo 7 > /sys/kernel/debug/omap_mux/gpmc_a0 && # RST
echo 7 > /sys/kernel/debug/omap_mux/uart0_ctsn && # MOSI
echo 3f > /sys/kernel/debug/omap_mux/uart0_rtsn && # MISO
echo 7 >  /sys/kernel/debug/omap_mux/mii1_col && # SCK
echo 7 > /sys/kernel/debug/omap_mux/mcasp0_ahclkr && # nCS
echo 113 > /sys/class/gpio/export &&
echo out > /sys/class/gpio/gpio113/direction &&
echo 0 > /sys/class/gpio/gpio113/value &&
${AVR_DUDE_BIN:-/usr/bin/avrdude} -p atmega328 -c ledmatrix -P ledmatrix -v -U flash:w:matrix_driver.hex -U lfuse:w:0xaf:m -U hfuse:w:0xd9:m -F -C avrduderc
rc=$?

echo in > /sys/class/gpio/gpio113/direction &&
echo 113 > /sys/class/gpio/unexport &&
echo 3f > /sys/kernel/debug/omap_mux/mcasp0_ahclkr # nCS
postrc=$?

$SKIP_SERVICE_CONTROL || start sphere-leds || true
$SKIP_SERVICE_CONTROL || start devkit-status-led || true

test $rc -eq 0 && $postrc -eq 0