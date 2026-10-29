// Plays ANTS, the computer against itself, without the display (as the
// demo of the open screen does), and prints a CRC-32 of the state after
// each turn every 100 turns (ants_test.go writes the same lines).
#include "include.h"
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
	char line[512];
	int wins[2] = { 0, 0 };
	RandInitSeed(6);
	NewGame(PLAYER_COMP, PLAYER_COMP);
	for (int t = 0; t < 2000; t++) {
		EndGame = -1;
		AutoPlay();
		if (EndGame >= 0) wins[EndGame]++;
		int n = sprintf(line, "%d: p%d last %d%s", t, Player, Last, LastDiscard ? "d" : "");
		for (int i = 0; i < 2; i++) {
			n += sprintf(line+n, " |");
			for (int j = 0; j < PAR_NUM; j++) n += sprintf(line+n, " %d", Players[i].par[j]);
			n += sprintf(line+n, " :");
			for (int j = 0; j < CARDNUM; j++) n += sprintf(line+n, " %d", Players[i].cards[j]);
		}
		sprintf(line+n, "\n");
		crcAdd(line);
		if (t % 100 == 99) printf("%d %08x\n", t, ~crc);
	}
	printf("wins %d %d\n", wins[0], wins[1]);
	return 0;
}
