#!/bin/sh
# Compiles the C source of PicoLibSDK's RAPTOR on the host, with stubs
# for the SDK (include.h), and plays it with a script (sim.cpp): the
# checksums TestMatchesC compares with.
#
#	PICOLIBSDK=~/PicoLibSDK sh games/raptor/testdata/csim/run.sh
set -e
dir=$(cd "$(dirname "$0")" && pwd)
tmp=$(mktemp -d)
mkdir "$tmp/src"
for f in main.cpp main.h; do
	tr -d '\r' < "$PICOLIBSDK/PicoPad/GAME/RAPTOR/src/$f" > "$tmp/src/$f"
done
cp "$dir/include.h" "$dir/sim.cpp" "$tmp/"
cd "$tmp"
g++ -O2 -w -I. -o sim sim.cpp
./sim
rm -r "$tmp"
