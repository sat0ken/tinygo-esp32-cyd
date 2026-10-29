#pragma once
// Stubs to compile the CHESS main.cpp on the host (for perft).
#include <stdint.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
typedef uint8_t u8; typedef int8_t s8; typedef int16_t s16; typedef uint16_t u16; typedef uint32_t u32; typedef int Bool;
#define True 1
#define False 0
#define INLINE static inline
#define B4 (1<<4)
#define B5 (1<<5)
#define WIDTH 320
#define HEIGHT 240
#define COLOR(r,g,b) ((u16)((((r)&0xf8)<<8)|(((g)&0xfc)<<3)|(((b)&0xf8)>>3)))
#define COL_WHITE 0xffff
#define COL_BLACK 0
#define COL_YELLOW 0xffe0
#define NOKEY 0
#define KEY_UP 1
#define KEY_LEFT 2
#define KEY_RIGHT 3
#define KEY_DOWN 4
#define KEY_X 5
#define KEY_Y 6
#define KEY_A 7
#define KEY_B 8
typedef struct { char s[64]; } pText;
static const u8* pDrawFont; static int DrawFontHeight, DrawFontWidth;
static const u8 FontBold8x16[1] = {0}, FontBold8x8[1] = {0};
static void DrawImg4Pal(const u8* a, const u16* b, int c, int d, int e, int f, int g, int h, int i) {}
static void DrawRect(int x, int y, int w, int h, u16 c) {}
static void DrawText(const char* t, int x, int y, u16 c) {}
static void DrawText2(const char* t, int x, int y, u16 c) {}
static void DrawTextBg(const char* t, int x, int y, u16 c, u16 b) {}
static void DrawClear() {}
static void DispUpdate() {}
static void WaitMs(int ms) {}
static void VgaWaitVSync() {}
static void KeyFlush() {}
static u8 KeyGet() { return NOKEY; }
static u32 Time() { return 0; }
static void ResetToBootLoader() { exit(0); }
static int TextLen(const pText* t) { return strlen(t->s); }
static const char* TextPtr(const pText* t) { return t->s; }
static void TextInit(pText* t) { t->s[0] = 0; }
static void TextTerm(pText* t) {}
static void TextSetStr(pText* t, const char* s) { strcpy(t->s, s); }
static void TextEmpty(pText* t) { t->s[0] = 0; }
static void TextAddSpc(pText* t) { strcat(t->s, " "); }
static void TextAddCh(pText* t, char c) { int n = strlen(t->s); t->s[n] = c; t->s[n+1] = 0; }
static void TextAddStr(pText* t, const char* s) { strcat(t->s, s); }
static void TextAddUInt(pText* t, int v, int x) { sprintf(t->s + strlen(t->s), "%d", v); }
// the random numbers of lib_rand.c, seeded like picopad.New(seed)
static uint64_t RandSeedV;
static u32 RandShift() { RandSeedV = RandSeedV*214013 + 2531011; return (u32)(RandSeedV >> 32); }
static void RandInitSeed(u32 seed) { RandSeedV = 123456789123456789ULL + 198765432198765432ULL + seed; RandShift(); RandShift(); RandShift(); }
static u8 RandU8() { return (u8)(RandShift() >> 24); }
static u16 RandU16() { return (u16)(RandShift() >> 16); }
static u32 Mask(u32 v) { v |= v>>1; v |= v>>2; v |= v>>4; v |= v>>8; v |= v>>16; return v; }
static u8 RandU8Max(u8 max) { if (!max) return 0; u8 m = Mask(max); u8 r; do { r = RandU8() & m; } while (r > max); return r; }
static u16 RandU16Max(u16 max) { if (!max) return 0; u16 m = Mask(max); u16 r; do { r = RandU16() & m; } while (r > max); return r; }
static s16 RandS16Max(s16 max) { if (max >= 0) return (s16)RandU16Max((u16)max); return -(s16)RandU16Max((u16)-max); }

#include "src/main.h"
