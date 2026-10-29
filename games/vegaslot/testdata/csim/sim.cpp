// Plays VEGASLOT's reel choice without the display: 3000 games, the bet
// lines 1..5 in turn, every 7th a bonus game; prints a CRC-32 of the
// reel positions and prizes every 500 games, and the sums of the bets and
// the prizes (vegaslot_test.go does the same).
#include "include.h"
#define main vegaslot_main
#include "src/main.cpp"
#undef main
#include "src/Table.cpp"

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
	char line[128];
	long bets = 0, prizes = 0;
	RandInitSeed(3);
	for (int n = 0; n < 3000; n++) {
		int lines = n % 5 + 1;
		Bonus = (n % 7 == 6) ? 1000 : 0;
		GetReelPos(lines);
		u16 winsym = TestSymbol(Reels[0].dest/256, Reels[1].dest/256, Reels[2].dest/256);
		int prize = GetPrize(lines, winsym);
		if (Bonus == 0) { bets += lines; prizes += prize; }
		sprintf(line, "%d: %d %d %d %d\n", n, Reels[0].dest, Reels[1].dest, Reels[2].dest, prize);
		crcAdd(line);
		if (n % 500 == 499) printf("%d %08x\n", n, ~crc);
	}
	printf("bets %ld prizes %ld\n", bets, prizes);
	return 0;
}
