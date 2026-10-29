#!/bin/sh
# Compiles the C source of PicoLibSDK's ANTS on the host (as C), with
# stubs for the SDK (include.h), and plays the computer against itself
# (sim.c): the checksums TestMatchesC compares with.
#
#	PICOLIBSDK=~/PicoLibSDK sh games/ants/testdata/csim/run.sh
set -e
dir=$(cd "$(dirname "$0")" && pwd)
tmp=$(mktemp -d)
mkdir "$tmp/src"
for f in game.cpp game.h main.h; do
	tr -d '\r' < "$PICOLIBSDK/PicoPad/GAME/ANTS/src/$f" > "$tmp/src/$f"
done
cp "$dir/include.h" "$dir/sim.c" "$tmp/"
cd "$tmp"
gcc -O2 -w -I. -o sim sim.c -lm
./sim
rm -r "$tmp"
