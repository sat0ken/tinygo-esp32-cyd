// Plays PICTOR without drawing: the frames of Game, the scenes one after
// another (the start of a scene without the prompt), with the buttons of
// the script below (the same in pictor_test.go), the life back to 300
// below 50. Prints a CRC-32 of the state lines every 500 frames
// (TestMatchesC writes the same lines).
#include "include.h"
#include "src/actor.cpp"
#include "src/background.cpp"
#include "src/bonus.cpp"
#include "src/bullet.cpp"
#include "src/enemy.cpp"
#include "src/foot.cpp"
#include "src/game.cpp"
#include "src/gamestart.cpp"
#include "src/header.cpp"
#include "src/missile.cpp"
#include "src/point.cpp"
#include "src/random.cpp"
#include "src/sound.cpp"

Bool ReqScreenShot;


static int Frame;

static Bool KeyPressed(u8 key)
{
	switch (key) {
	case KEY_A: return Frame % 200 < 150;
	case KEY_LEFT: return Frame % 40 < 10;
	case KEY_RIGHT: return Frame % 40 >= 20 && Frame % 40 < 30;
	case KEY_UP: return Frame % 60 >= 30 && Frame % 60 < 45;
	case KEY_DOWN: return Frame % 60 >= 45 && Frame % 60 < 55;
	}
	return False;
}

static u8 KeyGet()
{
	if (Frame % 90 == 45) return KEY_B;
	if (Frame % 250 == 100) return KEY_X;
	return NOKEY;
}

// NewGame without GameStart
static void SimNewGame(int level)
{
	Level = level;
	CurrentFrame = 0;
	MaxFrame = GAMELEN;
	BloodTime = 0;
	EnemyRandSeed = Level + 123456;
	InitBack();
	InitActor();
	InitMissile();
	InitEnemy();
	InitBullet();
	InitBonus();
	InitPoint();
	DispGameStart(False);
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
	int i;
	RandInitSeed(5);
	Score = 0;
	BonusMask = 0;
	Life = LIFE_INIT;
	ActInx = 0;
	SimNewGame(0);
	for (Frame = 0; Frame < 9000; Frame++) {
		NewEnemy();
		CtrlActor();
		DispBack(); DispActor(); EnemyDisp(); ExplosionDisp(); BulletDisp(); MissileDisp();
		DispBlood(); BonusDisp(); PointDisp(); DispHeader(); DispFoot(); DispUpdate();
		EnemyShift();
		ExplosionShift();
		BulletShift();
		MissileShift();
		EnemyHit();
		ShiftBack();
		ShiftActor();
		ShiftBlood();
		BonusShift();
		PointShift();
		HitActor();
		BonusCollect();
		CurrentFrame++;
		if (CurrentFrame >= MaxFrame) SimNewGame(Level + 1);
		if (Life < 50) Life = 300;

		sprintf(line, "%d: f%d lv%d sc%d life%d a%d %d,%d ph%d j%d r%d bm%d b%d %d,%d,%d s%u bp%d,%d,%d bl%d %d,%d",
			Frame, CurrentFrame, Level, Score, Life, ActInx, ActorX, ActorY, ActorPhase, ActorJumpSpeed, ActorReload,
			BonusMask, BonusInx, BonusX, BonusY, BonusPhase, EnemyRandSeed, BackPhase1, BackPhase2, BackPhase3,
			BloodTime, BloodX, BloodY);
		crcAdd(line);
		for (i = 0; i < ENEMY_MAX; i++) {
			sEnemy* e = &Enemy[i];
			if (e->temp != NULL) { sprintf(line, " e%d:%d,%d,%d,%d,%d", i, e->inx, e->x, e->y, e->anim, e->step); crcAdd(line); }
		}
		for (i = 0; i < EXPLOSIONS_MAX; i++) {
			sExplosion* x = &Explosion[i];
			if (x->anim < EXPLOSION_PHASES) { sprintf(line, " x%d:%d,%d,%d", i, x->anim, x->x, x->y); crcAdd(line); }
		}
		for (i = 0; i < BULLET_MAX; i++) {
			sBullet* b = &Bullet[i];
			if (b->temp != NULL) { sprintf(line, " b%d:%d,%d,%d", i, (int)(b->temp - BulletTemp), b->x, b->y); crcAdd(line); }
		}
		for (i = 0; i < MISSILE_MAX; i++) {
			sMissile* m = &Missile[i];
			if (m->temp != NULL) {
				sprintf(line, " m%d:%d,%d,%d,%d,%d,%d,%d,%d,%d", i, m->actinx, m->x, m->y, m->dx, m->dy, m->phase,
					m->lifetime, m->hits, m->target == NULL ? -1 : (int)(m->target - Enemy));
				crcAdd(line);
			}
		}
		for (i = 0; i < POINT_NUM; i++) {
			sPoint* p = &Point[i];
			if (p->len > 0) { sprintf(line, " p%d:%s,%d,%d,%d", i, p->buf, p->x, p->y, p->frames); crcAdd(line); }
		}
		crcAdd("\n");
		if (Frame % 500 == 499) printf("%d %08x sc %d lv %d life %d\n", Frame, ~crc, Score, Level, Life);
	}
	return 0;
}
