//go:build esp32s3

// Package espmalloc provides __wrap_malloc/calloc/free for C code (cgo)
// when the Wi-Fi driver is not linked.
//
// TinyGo's esp32s3 target links with --wrap=malloc,calloc,free,realloc so
// that the espradio package can supply its own allocator. Without espradio,
// any C code that calls malloc (such as tinyfs/fatfs) fails to link with
// "undefined symbol: __wrap_malloc". These wrappers forward to the originals
// (__real_*), which are TinyGo's GC-backed malloc/calloc/free from
// runtime/baremetal.go. Import it for side effects:
//
//	import _ "github.com/sat0ken/tinygo-cyd/internal/espmalloc"
//
// Do not import it together with espradio (duplicate symbols).
package espmalloc

import "unsafe"

//export __real_malloc
func realMalloc(size uintptr) unsafe.Pointer

//export __real_calloc
func realCalloc(nmemb, size uintptr) unsafe.Pointer

//export __real_free
func realFree(ptr unsafe.Pointer)

//export __wrap_malloc
func wrapMalloc(size uintptr) unsafe.Pointer { return realMalloc(size) }

//export __wrap_calloc
func wrapCalloc(nmemb, size uintptr) unsafe.Pointer { return realCalloc(nmemb, size) }

//export __wrap_free
func wrapFree(ptr unsafe.Pointer) { realFree(ptr) }
