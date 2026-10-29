#!/bin/sh
# Compiles the C source of PicoLibSDK's PACMAN on the host, with stubs for
# the SDK (include.h), and plays it without a player (sim.cpp): the
# checksums TestMatchesC compares with.
#
#	PICOLIBSDK=~/PicoLibSDK sh games/pacman/testdata/csim/run.sh
set -e
dir=$(cd "$(dirname "$0")" && pwd)
tmp=$(mktemp -d)
mkdir "$tmp/src"
for f in data.cpp game.cpp def.h data.h game.h; do
	tr -d '\r' < "$PICOLIBSDK/PicoPad/GAME/PACMAN/src/$f" > "$tmp/src/$f"
done
cp "$dir/include.h" "$dir/sim.cpp" "$tmp/"
cd "$tmp"
g++ -O2 -w -I. -o sim sim.cpp
./sim
rm -r "$tmp"
