package switcher

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa -framework ApplicationServices

#include <stdlib.h>
#include <sys/types.h>
extern char* run_applescript_sync(const char* source);
extern int check_accessibility_trusted(void);
extern void prompt_accessibility_if_needed(void);
extern int raise_window(pid_t appPID, const char* ttyMatch, const char* titleMatch);
extern pid_t find_app_pid(const char* appName);
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

// runAppleScript executes AppleScript in-process via NSAppleScript.
func runAppleScript(source string) (string, error) {
	cs := C.CString(source)
	defer C.free(unsafe.Pointer(cs))
	result := C.run_applescript_sync(cs)
	defer C.free(unsafe.Pointer(result))
	s := C.GoString(result)
	if strings.HasPrefix(s, "error:") {
		return "", fmt.Errorf("AppleScript: %s", s[6:])
	}
	return s, nil
}

func isAccessibilityTrusted() bool {
	return C.check_accessibility_trusted() == 1
}

func PromptAccessibility() {
	C.prompt_accessibility_if_needed()
}

// raiseWindow uses AX C API to find and raise a specific window.
// Returns: 1=matched by tty, 2=matched by title, 0=not found, -1=not trusted
func raiseWindow(appPID int, ttyDevice, titleMatch string) int {
	var ctty, ctitle *C.char
	if ttyDevice != "" {
		ctty = C.CString(ttyDevice)
		defer C.free(unsafe.Pointer(ctty))
	}
	if titleMatch != "" {
		ctitle = C.CString(titleMatch)
		defer C.free(unsafe.Pointer(ctitle))
	}
	return int(C.raise_window(C.int(appPID), ctty, ctitle))
}

func findAppPID(name string) int {
	cs := C.CString(name)
	defer C.free(unsafe.Pointer(cs))
	return int(C.find_app_pid(cs))
}
