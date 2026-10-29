#pragma once
// Stubs to compile the RAPTOR main.cpp on the host (sim.cpp): the drawing
// and the sounds do nothing, Time goes 1ms a call, the buttons are
// pressed by the script of sim.cpp.
#include <stdint.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <math.h>
// the float functions in double, rounded to float as the Go port does (the
// float versions of the C library round some results the other way)
#define atan2f(y, x) ((float)atan2((double)(y), (double)(x)))
#define cosf(a) ((float)cos((double)(a)))
#define sinf(a) ((float)sin((double)(a)))
#define sqrtf(a) ((float)sqrt((double)(a)))
typedef uint8_t u8; typedef int8_t s8; typedef int16_t s16; typedef uint16_t u16; typedef uint32_t u32; typedef int Bool;
#define True 1
#define False 0
#define WIDTH 320
#define HEIGHT 240
#define COLOR(r,g,b) ((u16)((((r)&0xf8)<<8)|(((g)&0xfc)<<3)|(((b)&0xf8)>>3)))
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
#define DrawImg4Pal(...) ((void)0)
#define DrawBlitPal(...) ((void)0)
#define DrawBlit4Pal(...) ((void)0)
#define DrawBlit1Shadow(...) ((void)0)
#define PlaySoundChan(...) ((void)0)
static void DispUpdate() {}
static void DispSetStripNext() {}
static void StopSoundChan(int c) {}
static Bool PlayingSoundChan(int c) { return False; }
static void ResetToBootLoader() { exit(0); }
static u8 KeyGet() { return NOKEY; }
static Bool KeyPressed(u8 key);
static u32 FakeTime;
static u32 Time() { return FakeTime += 1000; }
// the random numbers of lib_rand.c, seeded like picopad.New(seed)
static uint64_t RandSeedV;
static u32 RandShift() { RandSeedV = RandSeedV*214013 + 2531011; return (u32)(RandSeedV >> 32); }
static void RandInitSeed(u32 seed) { RandSeedV = 123456789123456789ULL + 198765432198765432ULL + seed; RandShift(); RandShift(); RandShift(); }
static u8 RandU8() { return (u8)(RandShift() >> 24); }
static u16 RandU16() { return (u16)(RandShift() >> 16); }
static u32 Mask(u32 v) { v |= v>>1; v |= v>>2; v |= v>>4; v |= v>>8; v |= v>>16; return v; }
static u8 RandU8Max(u8 max) { if (!max) return 0; u8 m = Mask(max); u8 r; do { r = RandU8() & m; } while (r > max); return r; }
static u16 RandU16Max(u16 max) { if (!max) return 0; u16 m = Mask(max); u16 r; do { r = RandU16() & m; } while (r > max); return r; }
static u16 RandU16MinMax(u16 min, u16 max) { return RandU16Max(max - min) + min; }
static float RandFloat() { return (float)(RandShift() >> (32-24))*(1.0f/0x1000000); }
static float RandFloatMinMax(float min, float max) { return RandFloat()*(max - min) + min; }

#include "src/main.h"
