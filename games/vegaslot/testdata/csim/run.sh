#!/bin/sh
# Compiles the C source of PicoLibSDK's VEGASLOT on the host, with stubs
# for the SDK (include.h), and plays the reel choice (sim.cpp): the
# checksums TestMatchesC compares with.
#
#	PICOLIBSDK=~/PicoLibSDK sh games/vegaslot/testdata/csim/run.sh
set -e
dir=$(cd "$(dirname "$0")" && pwd)
tmp=$(mktemp -d)
mkdir "$tmp/src"
for f in main.cpp main.h Table.cpp; do
	tr -d '\r' < "$PICOLIBSDK/PicoPad/GAME/VEGASLOT/src/$f" > "$tmp/src/$f"
done
cp "$dir/include.h" "$dir/sim.cpp" "$tmp/"
cd "$tmp"
g++ -O2 -w -I. -o sim sim.cpp
./sim
rm -r "$tmp"
