//go:build esp32_3248s035c

package platform

import (
	"runtime/volatile"
	"unsafe"
)

// Check of the flash MMU entries of the program code (IROM).
//
// cmd/games did not reach main on the board, while every smaller program
// worked. It is the only program whose code is longer than one 64KB flash
// page, and main.main is in the second page (0x400E0000..). TinyGo's
// startup code (src/device/esp/esp32.S) maps the IROM pages with the ROM
// function cache_flash_mmu_set. This init prints the PRO CPU MMU entries
// of the DROM and IROM pages and the values they should have, and writes
// any wrong entry again (with the cache disabled, from IRAM).
//
// Layout (targets/esp32.ld, builder): DROM (.rodata) at 0x3F400000 from
// flash _drom_flash_addr, IROM (.text) at 0x400D0000 from the next 64KB
// page after the DROM pages. ESP32 MMU table of the PRO CPU: 0x3FF10000,
// entry n (n = 0..63) for DROM 0x3F400000 + n*64KB, entry 64+n for
// 0x40000000 + n*64KB; value = flash page number, 0x100 = invalid.

const (
	proMMUTable = 0x3FF10000
	mmuPage     = 0x10000
	iromVaddr   = 0x400D0000
	iromEntry0  = 64 + (iromVaddr-0x40000000)/mmuPage // 77
	mmuInvalid  = 0x100
	maxIROM     = 16 // pages checked at most
)

//go:extern _text_start
var textStart [0]byte

//go:extern _text_end
var textEnd [0]byte

//go:extern _rodata_start
var rodataStart [0]byte

//go:extern _rodata_end
var rodataEnd [0]byte

//go:extern _drom_flash_addr
var dromFlashAddr uint32

// ESP32 ROM functions (addresses in targets/esp32-3248s035c.json, from
// TinyGo's esp32.S).

//export Cache_Read_Disable
func cacheReadDisable(cpu uint32)

//export Cache_Read_Enable
func cacheReadEnable(cpu uint32)

//export Cache_Flush
func cacheFlush(cpu uint32)

func pages(start, end unsafe.Pointer) uint32 {
	n := uint32(uintptr(end) - uintptr(start))
	if n == 0 {
		return 0
	}
	return (n-1)/mmuPage + 1
}

func mmuEntry(i uint32) *uint32 {
	return (*uint32)(unsafe.Pointer(uintptr(proMMUTable + i*4)))
}

// iromWant holds the expected entries for fixIROM (IRAM code must not
// touch anything that could need the cache).
var iromWant [maxIROM]uint32

func init() {
	dromPages := pages(unsafe.Pointer(&rodataStart), unsafe.Pointer(&rodataEnd))
	iromPages := pages(unsafe.Pointer(&textStart), unsafe.Pointer(&textEnd))
	if iromPages > maxIROM {
		iromPages = maxIROM
	}
	dromFirst := dromFlashAddr / mmuPage
	iromFirst := dromFirst + dromPages
	println("boot: DROM", dromPages, "page(s) from flash page", dromFirst, "- MMU entry 0 =", volatile.LoadUint32(mmuEntry(0)))
	bad := uint32(0)
	for k := uint32(0); k < iromPages; k++ {
		got := volatile.LoadUint32(mmuEntry(iromEntry0 + k))
		iromWant[k] = iromFirst + k
		ok := "ok"
		if got != iromWant[k] {
			ok = "WRONG"
			bad++
		}
		println("boot: IROM page", k, "MMU entry", iromEntry0+k, "=", got, "want", iromWant[k], ok)
	}
	if bad > 0 {
		fixIROM(iromPages)
		for k := uint32(0); k < iromPages; k++ {
			println("boot: IROM page", k, "MMU entry now", volatile.LoadUint32(mmuEntry(iromEntry0+k)))
		}
	}
}

// fixIROM writes the IROM entries again. It runs from IRAM because the
// flash cache is off while the table changes.
//
//go:section .iram1.s035irom
//go:noinline
func fixIROM(n uint32) {
	cacheReadDisable(0)
	cacheFlush(0)
	for k := uint32(0); k < n; k++ {
		volatile.StoreUint32((*uint32)(unsafe.Pointer(uintptr(proMMUTable+(iromEntry0+k)*4))), iromWant[k])
	}
	cacheReadEnable(0)
}
