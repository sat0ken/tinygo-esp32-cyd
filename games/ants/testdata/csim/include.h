#pragma once
// Stubs to compile the ANTS game.cpp on the host as C (sim.c): the
// drawing, the sounds and the USB port do nothing, Time goes 1ms a call.
#include <stdint.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <math.h>
typedef uint8_t u8; typedef int8_t s8; typedef int16_t s16; typedef uint16_t u16; typedef uint32_t u32; typedef uint64_t u64; typedef int Bool;
#define True 1
#define False 0
#define true 1
#define WIDTH 320
#define HEIGHT 240
#define PI 3.14159265358979324
#define B1 (1<<1)
#define BIT(n) (1u<<(n))
#define COLOR(r,g,b) ((u16)((((r)&0xf8)<<8)|(((g)&0xfc)<<3)|(((b)&0xf8)>>3)))
#define COL_WHITE 0xffff
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
#define DISP_STRIP_NUM 1
#define DrawImg4Pal(...) ((void)0)
#define DrawImgPal(...) ((void)0)
#define DrawBlitPal(...) ((void)0)
#define DrawBlit4Pal(...) ((void)0)
#define DrawBlit1Shadow(...) ((void)0)
#define DrawImg(...) ((void)0)
#define DrawText(...) ((void)0)
#define DrawText2(...) ((void)0)
#define PLAYSOUND(s) ((void)0)
#define PLAYSOUNDREP(s) ((void)0)
#define PlaySound(s, n) ((void)0)
// the float functions in double, rounded to float as the Go port does
#define powf(a, b) ((float)pow((double)(a), (double)(b)))
static u16 FrameBuf[WIDTH*HEIGHT+WIDTH*64];
static const u8* pDrawFont; static int DrawFontHeight, DrawFontWidth;
static const u8 FontCond6x8[1], FontBold8x8[1];
static void SelFont8x16() {}
static void DrawClear() {}
static void DispUpdate() {}
static void DispSetStripNext() {}
static void DispSetStripOff() {}
static void VgaWaitVSync() {}
static void StopSound() {}
static Bool PlayingSound() { return False; }
static void WaitMs(int ms) {}
static void KeyFlush() {}
static u8 KeyGet() { return NOKEY; }
static u32 FakeTime;
static u32 Time() { return FakeTime += 1000; }
static int DecNum(char* buf, int n, char sep) { return sprintf(buf, "%d", n); }
// the random numbers of lib_rand.c, seeded like picopad.New(seed)
static u64 RandSeed[1];
static u32 RandShift() { RandSeed[0] = RandSeed[0]*214013 + 2531011; return (u32)(RandSeed[0] >> 32); }
static void RandInitSeed(u32 seed) { RandSeed[0] = 123456789123456789ULL + 198765432198765432ULL + seed; RandShift(); RandShift(); RandShift(); }
static u8 RandU8() { return (u8)(RandShift() >> 24); }
static float RandFloat() { return (float)(RandShift() >> (32-24))*(1.0f/0x1000000); }
// the USB Mini-Port of the SDK (not used: no USB player)
#define USBPORT_CMD_APP 16
#define USBPORT_CMD_HELLO 1
#define USBPORT_CMD_REQ 2
#define USBPORT_CMD_SEED 3
#define USBPORT_CMD_QUIT 4
typedef struct { u16 crc; u8 len, cmd; u32 uid; u16 ver, var; u8 infolen; char info[10]; } sUsbPortPktHello;
typedef struct { u16 crc; u8 len, cmd; } sUsbPortPktQuit;
typedef struct { u16 crc; u8 len, cmd; u32 req; } sUsbPortPktReq;
typedef struct { u16 crc; u8 len, cmd; u32 seedL, seedH; } sUsbPortPktSeed;
typedef struct { u16 crc; u8 len, cmd; u8 data[60]; } sUsbPortPkt;
static Bool UsbPortSendReady() { return True; }
static Bool UsbPortSend(const void* p) { return True; }
static Bool UsbPortRecv(void* p) { return False; }
static Bool UsbPortConnected() { return True; }
static Bool UsbPortIsMaster() { return True; }
static void UsbPortInit(int m) {}
static void UsbPortTerm() {}

#include "src/main.h"
#include "src/game.h"
