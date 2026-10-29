// Plays ZOOM without drawing: the frames of Game, 30 ms each, with a
// player who flies at the nearest gate (or else at the nearest enemy),
// shoots all the time and changes the speed (the same in zoom_test.go);
// 3 lives again when they are gone. Prints a CRC-32 of the state lines
// every 1000 frames (TestMatchesC writes the same lines).
#include "include.h"
#include "src/enemy.cpp"
#include "src/fuel.cpp"
#include "src/gates.cpp"
#include "src/goals.cpp"
#include "src/ground.cpp"
#include "src/lives.cpp"
#include "src/score.cpp"
#include "src/ship.cpp"
#include "src/shot.cpp"
#include "src/stars.cpp"
#include "src/open.cpp"
#define main zoom_main
#include "src/main.cpp"
#undef main

static int Frame;
static Bool Left, Right;

static Bool KeyPressed(u8 key)
{
	switch (key) {
	case KEY_A: return True;
	case KEY_LEFT: return Left;
	case KEY_RIGHT: return Right;
	case KEY_UP: return Frame % 300 < 100;
	case KEY_DOWN: return Frame % 300 >= 200 && Frame % 300 < 260;
	}
	return False;
}

static u8 KeyGet() { return NOKEY; }

// the player: to the nearest gate ahead, or else to the nearest enemy
static void Decide()
{
	int target = -1, best = 1 << 30, i;
	for (i = 0; i < GATES; i++) {
		sGate* g = &Gates[i];
		if (g->x != GATEFREE && !g->def && g->z > 0 && g->z < best) { best = g->z; target = g->x << FRAC; }
	}
	if (target < 0) {
		for (i = 0; i < OBJ_MAX; i++) {
			sObj* o = &Obj[i];
			if ((o->type == ENEMY_DISK || o->type == ENEMY_BIG || o->type == ENEMY_BOSS) && o->z > 0 && o->z < best) { best = o->z; target = o->x; }
		}
	}
	Left = target >= 0 && target < ShipX - (4 << FRAC);
	Right = target >= 0 && target > ShipX + (4 << FRAC);
}

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
	int i, dz;
	u32 dt = 30000;
	RandInitSeed(9);
	CalcPerspTab();
	NewGame();
	InitLevel();
	for (Frame = 0; Frame < 30000; Frame++) {
		Decide();
		DispAll();
		if (SubLevel == 3) { ShiftStars(); NewStars(); }
		// Game after the time of the frame
		if (SubLevel == 3)
			dz = (dt*(SHIP_SPEEDDEF>>FRAC))>>(21-FRAC);
		else
			dz = (dt*(Speed>>FRAC))>>(21-FRAC);
		AddScore(dz);
		SubFuel(dt>>13);
		GrassPhase -= dz;
		while (GrassPhase < 0) GrassPhase += GROUND_DY<<FRAC;
		DoObj();
		ShipEnemyCrash();
		DoShip(dt, dz);
		Shooting(dt);
		Gating(dz);
		if ((Goals <= 0) && ((int)(Time() - GoalTime) > 1000000)) {
			SubLevel++;
			if (SubLevel > 3) { SubLevel = 0; Level++; }
			InitLevel();
		}
		FakeTime += dt;
		if (Lives == 0) Lives = 3;

		sprintf(line, "%d: t%u l%d.%d sc%d li%d f%d g%d s%d,%d c%d gp%d bl%d",
			Frame, Time(), Level, SubLevel, Score, Lives, Fuel, Goals, ShipX, Speed, ShipInCrash, GrassPhase, BossLives);
		crcAdd(line);
		for (i = 0; i < OBJ_MAX; i++) {
			sObj* o = &Obj[i];
			if (o->type != ENEMY_NO) { sprintf(line, " o%d:%d,%d,%d,%d,%d,%d,%d", i, o->type, o->move, o->istep, o->x, o->z, o->sx, o->sz); crcAdd(line); }
		}
		for (i = 0; i < SHOTS; i++)
			if (Shots[i].x != SHOTFREE) { sprintf(line, " s%d:%d,%d", i, Shots[i].x, Shots[i].z); crcAdd(line); }
		for (i = 0; i < GATES; i++)
			if (Gates[i].x != GATEFREE) { sprintf(line, " g%d:%d,%d,%d,%d", i, Gates[i].x, Gates[i].z, Gates[i].def, Gates[i].bomb); crcAdd(line); }
		for (i = 0; i < STARS; i++)
			if (Stars[i].x != STARFREE) { sprintf(line, " *%d:%d,%d,%d,%d", i, Stars[i].x, Stars[i].y, Stars[i].dx, Stars[i].dy); crcAdd(line); }
		crcAdd("\n");
		if (Frame % 1000 == 999) printf("%d %08x l%d.%d sc %d li %d\n", Frame, ~crc, Level, SubLevel, Score, Lives);
	}
	return 0;
}
