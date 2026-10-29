#pragma once
// Stubs to compile the ZOOM game on the host (sim.cpp): the drawing and
// the sounds do nothing, the time is FakeTime (WaitMs moves it on), the
// buttons are pressed by the script of sim.cpp.
#include <stdint.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <math.h>
// the float functions in double, rounded to float as the Go port does
#define sinf(a) ((float)sin((double)(a)))
#define cosf(a) ((float)cos((double)(a)))
typedef uint8_t u8; typedef int8_t s8; typedef int16_t s16; typedef uint16_t u16; typedef uint32_t u32; typedef int32_t s32; typedef int Bool;
typedef u16 COLTYPE;
#define True 1
#define False 0
#define WIDTH 320
#define HEIGHT 240
#define PI2 (3.14159265358979324*2)
#define COLOR(r,g,b) ((u16)((((r)&0xf8)<<8)|(((g)&0xfc)<<3)|(((b)&0xf8)>>3)))
#define COL_BLACK COLOR(0,0,0)
#define COL_WHITE COLOR(255,255,255)
#define COL_RED COLOR(255,0,0)
#define COL_GREEN COLOR(0,255,0)
#define COL_BLUE COLOR(0,0,255)
#define COL_YELLOW COLOR(255,255,0)
#define COL_CYAN COLOR(0,255,255)
#define COL_MAGENTA COLOR(255,0,255)
#define COL_LTGREEN COLOR(127,255,127)
#define COL_LTRED COLOR(255,127,127)
#define COL_LTYELLOW COLOR(255,255,127)
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
#define SNDFORM_PCM 0
#define count_of(a) (sizeof(a)/sizeof((a)[0]))
#define DrawRect(...) ((void)0)
#define DrawClear(...) ((void)0)
#define DrawText(...) ((void)0)
#define DrawTextBg(...) ((void)0)
#define DrawPoint(...) ((void)0)
#define DrawBlitPal(...) ((void)0)
#define DrawBlit4Pal(...) ((void)0)
#define DrawImgRle(...) ((void)0)
#define PlaySoundChan(...) ((void)0)
#define PLAYSOUND(snd) ((void)0)
#define MemPrint(buf, n, ...) snprintf(buf, n, __VA_ARGS__)
static void DispUpdate() {}
static void DispSetStripNext() {}
static void DispSetStripOff() {}
static void StopAllSound() {}
static void StopSoundChan(int c) {}
static void ResetToBootLoader() { exit(0); }
static u32 FakeTime;
static u32 Time() { return FakeTime; }
static void WaitMs(int ms) { FakeTime += ms*1000; }
static void WaitUs(int us) { FakeTime += us; }
static void KeyFlush() {}
static u8 KeyGet();
static Bool KeyPressed(u8 key);
#define DECNUMBUF_SIZE 16
static char DecNumBuf[DECNUMBUF_SIZE];
static int DecNum(char* buf, int val, char sep) { return sprintf(buf, "%d", val); }
// the random numbers of lib_rand.c, seeded like picopad.New(seed)
static uint64_t RandSeedV;
static u32 RandShift() { RandSeedV = RandSeedV*214013 + 2531011; return (u32)(RandSeedV >> 32); }
static void RandInitSeed(u32 seed) { RandSeedV = 123456789123456789ULL + 198765432198765432ULL + seed; RandShift(); RandShift(); RandShift(); }
static u8 RandU8() { return (u8)(RandShift() >> 24); }
static u16 RandU16() { return (u16)(RandShift() >> 16); }
static u32 Mask(u32 v) { v |= v>>1; v |= v>>2; v |= v>>4; v |= v>>8; v |= v>>16; return v; }
static u8 RandU8Max(u8 max) { if (!max) return 0; u8 m = Mask(max); u8 r; do { r = RandU8() & m; } while (r > max); return r; }
static u16 RandU16Max(u16 max) { if (!max) return 0; u16 m = Mask(max); u16 r; do { r = RandU16() & m; } while (r > max); return r; }
static u8 RandU8MinMax(u8 min, u8 max) { return RandU8Max(max - min) + min; }
static u16 RandU16MinMax(u16 min, u16 max) { return RandU16Max(max - min) + min; }
static s16 RandS16MinMax(s16 min, s16 max) { return (s16)RandU16Max(max - min) + min; }
static float RandFloat() { return (float)(RandShift() >> (32-24))*(1.0f/0x1000000); }
static float RandFloatMax(float max) { return RandFloat()*max; }

#include "src/main.h"
#include "src/enemy.h"
#include "src/fuel.h"
#include "src/gates.h"
#include "src/goals.h"
#include "src/ground.h"
#include "src/lives.h"
#include "src/open.h"
#include "src/score.h"
#include "src/ship.h"
#include "src/shot.h"
#include "src/stars.h"
