#!/bin/sh
# Compiles the C source of PicoLibSDK's ZOOM on the host, with stubs for
# the SDK (include.h), and plays it with a script (sim.cpp): the checksums
# TestMatchesC compares with.
#
#	PICOLIBSDK=~/PicoLibSDK sh games/zoom/testdata/csim/run.sh
set -e
dir=$(cd "$(dirname "$0")" && pwd)
src=$PICOLIBSDK/PicoPad/GAME/ZOOM
tmp=$(mktemp -d)
mkdir "$tmp/src" "$tmp/img" "$tmp/snd"
for f in "$src"/src/*.cpp "$src"/src/*.h; do
	tr -d '\r' < "$f" > "$tmp/src/$(basename "$f")"
done
for f in "$src"/img/*.cpp "$src"/snd/*.cpp; do
	d=$(basename "$(dirname "$f")")
	tr -d '\r' < "$f" > "$tmp/$d/$(basename "$f")"
done
cp "$dir/include.h" "$dir/sim.cpp" "$tmp/"
cd "$tmp"
clang++ -O1 -w -I. -o sim sim.cpp img/*.cpp snd/*.cpp
./sim
rm -r "$tmp"
