#!/bin/sh
# Compiles the C source of PicoLibSDK's CHESS on the host, with stubs for
# the SDK (include.h), and prints the perft numbers of the start position
# and the computer's moves against itself with the random numbers of
# picopad.New(seed 7): the values TestPerft and TestComputerMatchesC
# compare with.
#
#	PICOLIBSDK=~/PicoLibSDK sh games/chess/testdata/cperft/run.sh
set -e
dir=$(cd "$(dirname "$0")" && pwd)
tmp=$(mktemp -d)
mkdir "$tmp/src"
tr -d '\r' < "$PICOLIBSDK/PicoPad/GAME/CHESS/src/main.cpp" > "$tmp/src/main.cpp"
tr -d '\r' < "$PICOLIBSDK/PicoPad/GAME/CHESS/src/main.h" > "$tmp/src/main.h"
cp "$dir/include.h" "$dir/perft.cpp" "$dir/pieces_stub.c" "$tmp/"
cd "$tmp"
g++ -O2 -w -I. -o perft perft.cpp pieces_stub.c
./perft
rm -r "$tmp"
