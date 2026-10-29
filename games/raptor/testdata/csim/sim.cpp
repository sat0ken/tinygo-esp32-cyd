// Plays RAPTOR without drawing: the frames of main, A held, LEFT for
// frames 40n..40n+9, RIGHT for 40n+20..40n+29, UP for 120n+60..120n+69,
// DOWN for 120n+100..120n+109, 50 shields. Prints a CRC-32 of the state lines every
// 500 frames (raptor_test.go writes the same lines).
#include "include.h"
#define main raptor_main
#include "src/main.cpp"
#undef main

static int Frame;

static Bool KeyPressed(u8 key)
{
	switch (key) {
	case KEY_A: return True;
	case KEY_LEFT: return Frame % 40 < 10;
	case KEY_RIGHT: return Frame % 40 >= 20 && Frame % 40 < 30;
	case KEY_UP: return Frame % 120 >= 60 && Frame % 120 < 70;
	case KEY_DOWN: return Frame % 120 >= 100 && Frame % 120 < 110;
	}
	return False;
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
	char line[4096];
	int i;
	RandInitSeed(4);
	NewGame();
	Shields = 50; // to see the later levels and weapons
	for (Frame = 0; Frame < 10000; Frame++) {
		DoLandscape();
		DoShip();
		DoShooting();
		MoveMissile();
		DoEnemy();
		DoEnemyShooting();
		MoveEnemyMissile();
		// the display (it ends the hits and the explosions)
		DispLandscape();
		DispShipShadow();
		DispEnemyShadow();
		DispShooting();
		DispEnemy();
		DispShip();
		DispEnemyShooting();
		DispShields();
		DispScore();
		DispSelWeapon();
		DispEnergy();
		GameFrame++;

		int n = sprintf(line, "%d: ship %d,%d sh %d en %d sc %d lv %d w %d/%d", Frame, ShipX, ShipY, Shields, Energy, Score, Level, Weapon, WeaponMax);
		for (i = 0; i < ENEMY_MAX; i++)
			if (Enemy[i].type >= 0) n += sprintf(line+n, " e%d:%d,%.3f,%.3f,%d", i, Enemy[i].type, Enemy[i].x, Enemy[i].y, Enemy[i].energy);
		crcAdd(line);
		for (i = 0; i < MYMISSILE_MAX; i++)
			if (MyMissile[i].type >= 0) { sprintf(line, " m%d:%d,%.3f,%.3f", i, MyMissile[i].type, MyMissile[i].x, MyMissile[i].y); crcAdd(line); }
		for (i = 0; i < ENMISSILE_MAX; i++)
			if (EnMissile[i].type >= 0) { sprintf(line, " n%d:%d,%.3f,%.3f", i, EnMissile[i].type, EnMissile[i].x, EnMissile[i].y); crcAdd(line); }
		crcAdd("\n");
		if (Frame % 500 == 499) printf("%d %08x sc %d lv %d sh %d\n", Frame, ~crc, Score, Level, Shields);
	}
	return 0;
}
