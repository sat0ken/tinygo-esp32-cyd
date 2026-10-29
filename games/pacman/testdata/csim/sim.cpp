// Plays PACMAN without a player, the frames of OpenGame without the keys:
// while the ghosts are frightened, the pacman turns to the free tile next
// to it nearest to a frightened ghost; else to the first dot next to it
// (the directions from frame/50), else every 6 frames to (n*7 + n/3) % 4
// for frame 6n.
// It starts with 30 lives, and at frame 1500 one dot is left (the next one
// eaten ends the level). Prints a CRC-32 of the state lines every 100
// frames (pacman_test.go writes the same lines), and what happened.
#include "include.h"
#include "src/data.cpp"
#include "src/game.cpp"

static u32 crc = 0xffffffff;

static void crcAdd(const char* s)
{
	for (; *s; s++) {
		crc ^= (u8)*s;
		for (int k = 0; k < 8; k++) crc = (crc >> 1) ^ (0xEDB88320 & -(crc & 1));
	}
}

int main()
{
	char line[256];
	RandInitSeed(5);
	Score = 0; Level = 0; Lives = 30;
	int frighten = 0, eaten = 0, fruit = 0;
	NewGame(False);
	int f;
	for (f = 0; f < 4000; f++) {
		int xp = XTOMAP(Chars[CH_PACMAN].x), yp = YTOMAP(Chars[CH_PACMAN].y), best = -1;
		for (int k = 0; k < 4; k++) {
			int d = (k + f/50) & 3;
			u8 b = GetBoardDir(xp, yp, d);
			if ((b == TILE_DOT) || (b == TILE_PELLET) || (b == TILE_PELLETOFF)) { best = d; break; }
		}
		if (FrightenCnt > 0) {
			int bestdist = 1 << 30;
			for (int d = 0; d < 4; d++) {
				if (!CheckBoard(xp, yp, d, False)) continue;
				int x2 = ShiftX(xp, d), y2 = ShiftY(yp, d);
				for (int i = 1; i < 5; i++) {
					if (Chars[i].state != S_FRIGHTEN) continue;
					int dx = XTOMAP(Chars[i].x) - x2, dy = YTOMAP(Chars[i].y) - y2;
					if (dx*dx + dy*dy < bestdist) { bestdist = dx*dx + dy*dy; best = d; }
				}
			}
		}
		if (best >= 0) NextDir = best;
		else if (f % 6 == 0) NextDir = ((f/6)*7 + (f/6)/3) % 4;
		if (f == 1500) DotCnt = 1;
		Phase = (Phase+1) & 3;
		WakeUp();
		Unfrighten();
		if (FruitCnt > 0) { FruitCnt--; if (FruitCnt == 0) FruitSpriteImg = -1; }
		if (!Move()) {
			if (Lives < 0) break;
			NewGame(True);
		}
		DispAll(SPEED);
		if (DotCnt <= 0) { Level++; NewGame(False); }

		int n = sprintf(line, "%d:", f);
		for (int i = 0; i < 5; i++) n += sprintf(line+n, " %d,%d,%d,%d", Chars[i].x, Chars[i].y, Chars[i].dir, Chars[i].state);
		sprintf(line+n, " s=%u l=%d d=%d fr=%d lv=%d\n", Score, Lives, DotCnt, FrightenCnt, Level);
		crcAdd(line);
		for (int i = 1; i < 5; i++) { if (Chars[i].state == S_FRIGHTEN) frighten++; if (Chars[i].state == S_EATEN) eaten++; }
		if (FruitSpriteImg >= 0) fruit++;
		if (f % 100 == 99) printf("%d %08x\n", f, ~crc);
	}
	printf("frames with: frightened %d, eaten %d, fruit %d\n", frighten, eaten, fruit);
	printf("end %d %s", f, line);
	return 0;
}
