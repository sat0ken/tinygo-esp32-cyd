//go:build js && wasm

// Package wasmlcd shows the RGB565 frame buffer on an HTML canvas and turns
// pointer events on the canvas into touches.
//
// Like the real panel, which is refreshed continuously by DMA, the canvas is
// updated on every requestAnimationFrame. Frames where nothing was drawn are
// skipped (dirty flag).
package wasmlcd

import (
	"errors"
	"image/color"
	"strconv"
	"syscall/js"

	"github.com/sat0ken/tinygo-cyd/framebuf"
	"tinygo.org/x/drivers/pixel"
)

// Display is a hal.Display and hal.Touch backed by a <canvas>.
type Display struct {
	*framebuf.Buffer

	rgba    []byte   // RGBA copy of the frame buffer
	jsArr   js.Value // Uint8ClampedArray of the same size
	imgData js.Value // ImageData over jsArr
	canvas  js.Value
	ctx     js.Value
	dirty   bool

	touchX, touchY int16
	pressed        bool

	rafID   js.Value
	onRAF   js.Func
	onFlush js.Func
	flushFn string
	onDown  js.Func
	onMove  js.Func
	onUp    js.Func
}

// New attaches to the canvas with the given element id and starts updating it.
func New(canvasID string, w, h int16) (*Display, error) {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", canvasID)
	if canvas.IsNull() || canvas.IsUndefined() {
		return nil, errors.New("wasmlcd: canvas #" + canvasID + " not found")
	}
	canvas.Set("width", int(w))
	canvas.Set("height", int(h))
	n := int(w) * int(h)
	d := &Display{
		Buffer: framebuf.New(make([]uint16, n), w, h),
		rgba:   make([]byte, n*4),
		canvas: canvas,
		ctx:    canvas.Call("getContext", "2d"),
		dirty:  true,
	}
	d.jsArr = js.Global().Get("Uint8ClampedArray").New(n * 4)
	d.imgData = js.Global().Get("ImageData").New(d.jsArr, int(w), int(h))

	d.onRAF = js.FuncOf(func(this js.Value, args []js.Value) any {
		d.flush()
		d.rafID = js.Global().Call("requestAnimationFrame", d.onRAF)
		return nil
	})
	d.onDown = js.FuncOf(func(this js.Value, args []js.Value) any {
		ev := args[0]
		d.canvas.Call("setPointerCapture", ev.Get("pointerId"))
		d.pointer(ev)
		d.pressed = true
		ev.Call("preventDefault")
		return nil
	})
	d.onMove = js.FuncOf(func(this js.Value, args []js.Value) any {
		if d.pressed {
			d.pointer(args[0])
		}
		return nil
	})
	d.onUp = js.FuncOf(func(this js.Value, args []js.Value) any {
		d.pressed = false
		return nil
	})
	// window.wasmlcdFlush_<id>() copies the frame buffer to the canvas right
	// away. The headless check (make test-browser) calls it before reading
	// the canvas, because headless Chromium with virtual time does not run
	// animation frames while timers keep firing.
	d.onFlush = js.FuncOf(func(this js.Value, args []js.Value) any {
		d.flush()
		return nil
	})
	d.flushFn = "wasmlcdFlush_" + canvasID
	js.Global().Set(d.flushFn, d.onFlush)
	canvas.Call("addEventListener", "pointerdown", d.onDown)
	canvas.Call("addEventListener", "pointermove", d.onMove)
	canvas.Call("addEventListener", "pointerup", d.onUp)
	canvas.Call("addEventListener", "pointercancel", d.onUp)
	d.rafID = js.Global().Call("requestAnimationFrame", d.onRAF)
	return d, nil
}

// Close stops the updates, removes the listeners and releases the callbacks.
func (d *Display) Close() {
	js.Global().Call("cancelAnimationFrame", d.rafID)
	d.canvas.Call("removeEventListener", "pointerdown", d.onDown)
	d.canvas.Call("removeEventListener", "pointermove", d.onMove)
	d.canvas.Call("removeEventListener", "pointerup", d.onUp)
	d.canvas.Call("removeEventListener", "pointercancel", d.onUp)
	js.Global().Delete(d.flushFn)
	d.onRAF.Release()
	d.onFlush.Release()
	d.onDown.Release()
	d.onMove.Release()
	d.onUp.Release()
}

// pointer converts the event position to frame buffer coordinates. The
// canvas may be scaled with CSS, so use its on-screen rectangle.
func (d *Display) pointer(ev js.Value) {
	rect := d.canvas.Call("getBoundingClientRect")
	rw, rh := rect.Get("width").Float(), rect.Get("height").Float()
	if rw <= 0 || rh <= 0 {
		return
	}
	x := (ev.Get("clientX").Float() - rect.Get("left").Float()) * float64(d.W) / rw
	y := (ev.Get("clientY").Float() - rect.Get("top").Float()) * float64(d.H) / rh
	d.touchX = clamp(int16(x), d.W)
	d.touchY = clamp(int16(y), d.H)
}

func clamp(v, size int16) int16 {
	if v < 0 {
		return 0
	}
	if v >= size {
		return size - 1
	}
	return v
}

// flush converts RGB565 to RGBA (5/6 bits widened by copying the top bits
// into the low bits) and draws it on the canvas.
func (d *Display) flush() {
	if !d.dirty {
		return
	}
	d.dirty = false
	for i, v := range d.Pix {
		c := framebuf.ToRGBA(v)
		o := i * 4
		d.rgba[o+0] = c.R
		d.rgba[o+1] = c.G
		d.rgba[o+2] = c.B
		d.rgba[o+3] = 0xFF
	}
	js.CopyBytesToJS(d.jsArr, d.rgba)
	d.ctx.Call("putImageData", d.imgData, 0, 0)
}

// ReadTouch implements hal.Touch with the latest pointer state.
func (d *Display) ReadTouch() (x, y int16, pressed bool) {
	return d.touchX, d.touchY, d.pressed
}

// Display marks the buffer for the next animation frame.
func (d *Display) Display() error {
	d.dirty = true
	return nil
}

// The drawing methods wrap framebuf.Buffer to set the dirty flag.

func (d *Display) SetPixel(x, y int16, c color.RGBA) {
	d.Buffer.SetPixel(x, y, c)
	d.dirty = true
}

func (d *Display) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	d.dirty = true
	return d.Buffer.FillRectangle(x, y, w, h, c)
}

func (d *Display) FillScreen(c color.RGBA) {
	d.Buffer.FillScreen(c)
	d.dirty = true
}

func (d *Display) DrawFastHLine(x0, x1, y int16, c color.RGBA) {
	d.Buffer.DrawFastHLine(x0, x1, y, c)
	d.dirty = true
}

func (d *Display) DrawFastVLine(x, y0, y1 int16, c color.RGBA) {
	d.Buffer.DrawFastVLine(x, y0, y1, c)
	d.dirty = true
}

func (d *Display) DrawRGBBitmap(x, y int16, data []uint16, w, h int16) error {
	d.dirty = true
	return d.Buffer.DrawRGBBitmap(x, y, data, w, h)
}

func (d *Display) DrawRGBBitmap8(x, y int16, data []uint8, w, h int16) error {
	d.dirty = true
	return d.Buffer.DrawRGBBitmap8(x, y, data, w, h)
}

func (d *Display) DrawBitmap(x, y int16, bitmap pixel.Image[pixel.RGB565BE]) error {
	d.dirty = true
	return d.Buffer.DrawBitmap(x, y, bitmap)
}

// SetBrightness imitates the backlight with a CSS filter on the canvas.
func SetBrightness(canvasID string, percent int) {
	c := js.Global().Get("document").Call("getElementById", canvasID)
	if c.IsNull() || c.IsUndefined() {
		return
	}
	c.Get("style").Set("filter", "brightness("+strconv.Itoa(percent)+"%)")
}
