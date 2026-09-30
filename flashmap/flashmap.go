//go:build esp32s3

// Package flashmap maps a region of the SPI flash into the data bus so it
// can be read like memory, although it is not part of the program image.
//
// Why: TinyGo boots the ESP32-S3 without a second stage bootloader, so the
// ROM loader walks every segment of the program image. It refuses a large
// DROM (rodata) segment ("Invalid image block, can't boot" with a ~1MB
// go:embed), so big data such as images is written to its own flash region
// and mapped at run time instead, like ESP-IDF's esp_mmu_map / partition
// mmap.
//
// The ROM cache functions used here must be defined for the linker, see
// the ldflags of targets/esp32-4827s043.json:
//
//	rom_Cache_Suspend_DCache = 0x400018b4
//	Cache_Resume_DCache      = 0x400018c0
//	Cache_Invalidate_Addr    = 0x400016b0
//
// (ESP-IDF components/esp_rom/esp32s3/ld/esp32s3.rom.ld, v5.4).
package flashmap

import (
	"errors"
	"runtime/interrupt"
	"runtime/volatile"
	"unsafe"
)

const (
	pageSize = 0x10000    // SOC_MMU_PAGE_SIZE (64KB)
	dbusBase = 0x3C000000 // SOC_MMU_DBUS_VADDR_BASE
	mmuTable = 0x600C5000 // DR_REG_MMU_TABLE
	entries  = 512        // SOC_MMU_ENTRY_NUM

	// TinyGo's startup code (src/device/esp/esp32s3.S) splits the table
	// with Cache_Set_IDROM_MMU_Size(0x400, 0x400) and maps DROM with
	// entries 256..511, while ESP-IDF v5.4 (hal/esp32s3/include/hal/mmu_ll.h,
	// mmu_ll_get_entry_id) indexes the shared table with
	// (vaddr & 0x1FFFFFF) >> 16, i.e. entry 0.. for 0x3C000000. With
	// identity mapping (virtual page n = flash page n) both give the same
	// result for TinyGo's own code, so which one the hardware uses cannot be
	// told from it. Map writes the entries for both interpretations; the
	// extra ones are unused by anything else.
	splitDROMFirst = 256
)

// MMU entry value: flash page number | SOC_MMU_ACCESS_FLASH (0) |
// SOC_MMU_VALID (0); SOC_MMU_INVALID is BIT(14).
// Source: ESP-IDF soc/esp32s3/include/soc/ext_mem_defs.h, mmu_ll_write_entry.
const mmuInvalid = 1 << 14

var (
	ErrAlign = errors.New("flashmap: offset must be 64KB aligned")
	ErrRange = errors.New("flashmap: region outside the mappable 16MB")
)

//export rom_Cache_Suspend_DCache
func romCacheSuspendDCache() uint32

//export Cache_Resume_DCache
func cacheResumeDCache(autoload uint32)

//export Cache_Invalidate_Addr
func cacheInvalidateAddr(addr, size uint32)

// Map makes size bytes of flash at offset readable at 0x3C000000+offset
// and returns them. offset must be 64KB aligned; the region must lie in
// the first 16MB of flash. Mapping the same region again is harmless.
func Map(offset, size uint32) ([]byte, error) {
	if offset%pageSize != 0 {
		return nil, ErrAlign
	}
	first := offset / pageSize
	n := (size + pageSize - 1) / pageSize
	if size == 0 || first+n > splitDROMFirst {
		return nil, ErrRange
	}
	state := interrupt.Disable()
	mapPages(first, n)
	interrupt.Restore(state)
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(dbusBase+offset))), size), nil
}

// mapPages runs from IRAM: while the data cache is suspended, nothing may
// be read through the cache, including this code's own constants. The
// sequence follows ESP-IDF s_do_mapping (esp_mm/esp_mmu_map.c): stop the
// cache, write the entries, invalidate the range, start the cache again.
//
//go:section .iram1.flashmap
//go:noinline
func mapPages(first, n uint32) {
	autoload := romCacheSuspendDCache()
	for i := uint32(0); i < n; i++ {
		page := first + i
		volatile.StoreUint32((*uint32)(unsafe.Pointer(uintptr(mmuTable+page*4))), page)
		volatile.StoreUint32((*uint32)(unsafe.Pointer(uintptr(mmuTable+(splitDROMFirst+page)*4))), page)
	}
	cacheInvalidateAddr(dbusBase+first*pageSize, n*pageSize)
	cacheResumeDCache(autoload)
}

// Entry returns the raw MMU table entry i (for diagnostics).
func Entry(i int) uint32 {
	if i < 0 || i >= entries {
		return mmuInvalid
	}
	return volatile.LoadUint32((*uint32)(unsafe.Pointer(uintptr(mmuTable + i*4))))
}
