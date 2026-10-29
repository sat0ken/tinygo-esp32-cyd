#pragma once
// Stubs to compile the PICTOR game logic on the host (sim.cpp): the
// drawing and the sounds do nothing, the buttons are pressed by the
// script of sim.cpp.
#include <stdint.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <math.h>
// powf in double, rounded to float as the Go port does
#define powf(a, b) ((float)pow((double)(a), (double)(b)))
#define f2i(x) ((int)(x))
typedef uint8_t u8; typedef int8_t s8; typedef int16_t s16; typedef uint16_t u16; typedef uint32_t u32; typedef int32_t s32; typedef int Bool;
typedef u16 COLTYPE;
#define True 1
#define False 0
#define WIDTH 320
#define HEIGHT 240
#define COLOR(r,g,b) ((u16)((((r)&0xf8)<<8)|(((g)&0xfc)<<3)|(((b)&0xf8)>>3)))
#define COL_BLACK COLOR(0,0,0)
#define COL_WHITE COLOR(255,255,255)
#define COL_RED COLOR(255,0,0)
#define COL_GREEN COLOR(0,255,0)
#define COL_YELLOW COLOR(255,255,0)
#define COL_CYAN COLOR(0,255,255)
#define COL_MAGENTA COLOR(255,0,255)
#define COL_AZURE COLOR(0,127,255)
#define COL_DKBLUE COLOR(0,0,127)
#define COL_LTBLUE COLOR(127,127,255)
#define COL_DKGREEN COLOR(0,127,0)
#define B0 (1<<0)
#define B1 (1<<1)
#define B2 (1<<2)
#define B3 (1<<3)
#define B4 (1<<4)
#define BIT(n) (1<<(n))
#define NOKEY 0
#define KEY_UP 1
#define KEY_LEFT 2
#define KEY_RIGHT 3
#define KEY_DOWN 4
#define KEY_X 5
#define KEY_Y 6
#define KEY_A 7
#define KEY_B 8
#define DISP_STRIP_NUM 1
#define SNDFORM_ADPCM 0
#define SND_NUM 12
#define StrLen(s) ((int)strlen(s))
#define count_of(a) (sizeof(a)/sizeof((a)[0]))
#define DrawRect(...) ((void)0)
#define DrawFrame(...) ((void)0)
#define DrawClear(...) ((void)0)
#define DrawText(...) ((void)0)
#define DrawTextW(...) ((void)0)
#define DrawText2(...) ((void)0)
#define DrawTextBg(...) ((void)0)
#define DrawBlitPal(...) ((void)0)
#define DrawBlit4Pal(...) ((void)0)
#define DrawBlit1Shadow(...) ((void)0)
#define DrawImgPal(...) ((void)0)
#define DispDirtyRect(...) ((void)0)
#define PlaySoundChan(...) ((void)0)
#define MemPrint(buf, n, ...) snprintf(buf, n, __VA_ARGS__)
static void DrawImgPalDbl(const u8* src, const u16* pal, int xs, int ys, int xd, int yd, int w, int h, int ws) {}
static void DispShadow() {}
static void DispUpdate() {}
static void DispSetStripNext() {}
static void StopAllSound() {}
static void WaitMs(int ms) {}
static void KeyFlush() {}
static u8 KeyGet();
static Bool KeyPressed(u8 key);
static Bool GameMenu() { return True; }
#define DECNUMBUF_SIZE 16
static char DecNumBuf[DECNUMBUF_SIZE];
static int DecNum(char* buf, int val, char sep) { return sprintf(buf, "%d", val); }
// the random numbers of lib_rand.c, seeded like picopad.New(seed)
static uint64_t RandSeedV;
static u32 RandShift() { RandSeedV = RandSeedV*214013 + 2531011; return (u32)(RandSeedV >> 32); }
static void RandInitSeed(u32 seed) { RandSeedV = 123456789123456789ULL + 198765432198765432ULL + seed; RandShift(); RandShift(); RandShift(); }
static u16 RandU16() { return (u16)(RandShift() >> 16); }
static u32 Mask(u32 v) { v |= v>>1; v |= v>>2; v |= v>>4; v |= v>>8; v |= v>>16; return v; }
static u16 RandU16Max(u16 max) { if (!max) return 0; u16 m = Mask(max); u16 r; do { r = RandU16() & m; } while (r > max); return r; }
static s16 RandS16MinMax(s16 min, s16 max) { return (s16)RandU16Max(max - min) + min; }

#include "src/main.h"
#include "src/header.h"
#include "src/background.h"
#include "src/foot.h"
#include "src/actor.h"
#include "src/bullet.h"
#include "src/bonus.h"
#include "src/enemy.h"
#include "src/game.h"
#include "src/gamestart.h"
#include "src/missile.h"
#include "src/point.h"
#include "src/random.h"
#include "src/sound.h"
