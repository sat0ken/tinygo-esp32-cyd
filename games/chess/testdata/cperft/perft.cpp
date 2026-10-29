#define main chess_main
#include "src/main.cpp"
#undef main

// perft: leaf positions after depth plies, as MoveCompLevel walks them
static long perft(int depth)
{
	int level = MoveDeep;
	SearchMoves(0);
	int n = MoveStackNum[level];
	if (depth <= 1) return n;
	long total = 0;
	MoveDeep = level + 1;
	sMove* m = &MoveStack[MoveStackStart[level]];
	for (int i = 0; i < n; i++, m++)
	{
		MovePassantPiece[level+1] = 0;
		MovePassantPos[level+1] = 0;
		if ((m->srcpiece & (PIECEMASK | MOVEMASK)) == (PAWN | NOMOVING))
		{
			int dif = m->dst - m->src;
			if ((dif == 20) || (dif == -20))
			{
				MovePassantPiece[level+1] = m->dst;
				MovePassantPos[level+1] = m->src + dif/2;
			}
		}
		DoMove(m);
		Player ^= 1;
		int top = MoveStackTop;
		total += perft(depth - 1);
		MoveStackTop = top;
		Player ^= 1;
		UndoMove(m);
	}
	MoveDeep = level;
	return total;
}

int main(int argc, char** argv)
{
	// perft of the start position
	NewGame();
	for (int d = 1; d <= 4; d++)
	{
		MoveDeep = 0; MoveStackTop = 0;
		printf("perft %d: %ld\n", d, perft(d));
	}
	// the computer's moves of a game against itself, seed 7, levels 2 and 3
	for (int lv = 2; lv <= 3; lv++)
	{
		RandInitSeed(7);
		NewGame();
		DeepMax = lv;
		printf("level %d:", lv);
		for (int k = 0; k < 16; k++)
		{
			MoveComp();
			if (LastMove == NULL) { printf(" END"); break; }
			printf(" %d-%d", LastMove->src, LastMove->dst);
			Player ^= 1;
			if (Player == WHITE_PLAYER) MoveNum++;
		}
		printf("\n");
	}
	return 0;
}
