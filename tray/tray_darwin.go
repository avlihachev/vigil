package tray

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa

#include <stdlib.h>

void tray_init(const char* title, const char* tooltip);
void tray_set_title(const char* title);
void tray_remove(void);

// callbacks implemented in Go
extern void trayOnClick();
*/
import "C"

import "unsafe"

var clickHandler func()

func Init(title, tooltip string, onClick func()) {
	clickHandler = onClick

	ct := C.CString(title)
	defer C.free(unsafe.Pointer(ct))
	ctt := C.CString(tooltip)
	defer C.free(unsafe.Pointer(ctt))

	C.tray_init(ct, ctt)
}

func SetTitle(title string) {
	ct := C.CString(title)
	defer C.free(unsafe.Pointer(ct))
	C.tray_set_title(ct)
}

func Remove() {
	C.tray_remove()
}

//export trayOnClick
func trayOnClick() {
	if clickHandler != nil {
		go clickHandler()
	}
}
