#pragma once
// Stubs to compile the VEGASLOT main.cpp and Table.cpp on the host (sim.cpp):
// the drawing and the sounds do nothing, Time goes 1ms a call.
#include <stdint.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
typedef uint8_t u8; typedef int8_t s8; typedef int16_t s16; typedef uint16_t u16; typedef uint32_t u32; typedef int Bool;
#define True 1
#define False 0
#define ALIGNED
#define WIDTH 320
#define HEIGHT 240
#define COLOR(r,g,b) ((u16)((((r)&0xf8)<<8)|(((g)&0xfc)<<3)|(((b)&0xf8)>>3)))
#define COL_WHITE 0xffff
#define COL_BLACK 0
#define COL_YELLOW 0xffe0
#define COL_RED 0xf800
#define COL_GREEN 0x07e0
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
#define count_of(a) (sizeof(a)/sizeof((a)[0]))
#define SNDFORM_PCM 0
#define DrawImg4Pal(...) ((void)0)
#define DrawImgPal(...) ((void)0)
#define DrawImgRle(...) ((void)0)
#define DrawRectShadow(...) ((void)0)
#define PLAYSOUND(s) ((void)0)
#define PlaySoundChan(...) ((void)0)
static void DispUpdate() {}
static void VgaWaitVSync() {}
static void StopSoundChan(int c) {}
static int PlayingSound() { return 0; }
static void WaitMs(int ms) {}
static void KeyFlush() {}
static u8 KeyGet() { return NOKEY; }
static void ResetToBootLoader() { exit(0); }
static u32 FakeTime;
static u32 Time() { return FakeTime += 1000; }
// the random numbers of lib_rand.c, seeded like picopad.New(seed)
static uint64_t RandSeedV;
static u32 RandShift() { RandSeedV = RandSeedV*214013 + 2531011; return (u32)(RandSeedV >> 32); }
static void RandInitSeed(u32 seed) { RandSeedV = 123456789123456789ULL + 198765432198765432ULL + seed; RandShift(); RandShift(); RandShift(); }
static u16 RandU16() { return (u16)(RandShift() >> 16); }
static u32 Mask(u32 v) { v |= v>>1; v |= v>>2; v |= v>>4; v |= v>>8; v |= v>>16; return v; }
static u16 RandU16Max(u16 max) { if (!max) return 0; u16 m = Mask(max); u16 r; do { r = RandU16() & m; } while (r > max); return r; }

#include "src/main.h"
