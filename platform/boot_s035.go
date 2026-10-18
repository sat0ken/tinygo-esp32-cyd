//go:build esp32_3248s035c

package platform

import "device/esp"

// Boot diagnostics of the ESP32-3248S035C. TinyGo's kernel exception
// handler (targets/esp32-interrupts.S) saves the cause and the PC in RTC
// STORE0 (0x5556xxxx, xxxx = EXCCAUSE) and STORE1 before it resets the
// chip, so a crash that ended in a reset is printed on the next boot. The
// reset reason is printed when it is not a power-on (the RST button is a
// power-on reset too).

// Reset reasons (RTC_CNTL RESET_STATE, PRO CPU), from ESP-IDF
// esp32/rom/rtc.h.
var resetReasons = [...]string{
	1: "power on", 3: "software (system)", 4: "legacy watchdog", 5: "deep sleep",
	6: "SDIO", 7: "timer group 0 watchdog", 8: "timer group 1 watchdog",
	9: "RTC watchdog", 10: "intrusion", 11: "timer group watchdog (CPU)",
	12: "software (CPU)", 13: "RTC watchdog (CPU)", 14: "APP CPU",
	15: "brownout", 16: "RTC watchdog (RTC)",
}

func init() {
	reason := esp.RTC_CNTL.RESET_STATE.Get() & esp.RTC_CNTL_RESET_STATE_RESET_CAUSE_PROCPU_Msk
	if reason != 1 {
		name := "?"
		if int(reason) < len(resetReasons) && resetReasons[reason] != "" {
			name = resetReasons[reason]
		}
		println("boot: reset reason", reason, name)
	}
	if s0 := esp.RTC_CNTL.STORE0.Get(); s0&0xFFFF0000 == 0x55560000 {
		println("boot: last run ended with exception cause", s0&0xFFFF, "at PC", esp.RTC_CNTL.STORE1.Get())
		esp.RTC_CNTL.STORE0.Set(0)
	}
}
