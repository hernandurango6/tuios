//go:build windows

package input

import (
	"os"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestReadSystemClipboardImage(t *testing.T) {
	// Open clipboard to empty it and set a dummy DIB
	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		t.Skip("Failed to open clipboard, skipping test")
	}

	// Backup text if any
	var backupText string
	hasBackupText := false
	if handle, _, _ := procGetClipboardData.Call(cfUnicodeText); handle != 0 {
		if ptr, _, _ := procGlobalLock.Call(handle); ptr != 0 {
			backupText = windows.UTF16PtrToString((*uint16)(unsafe.Pointer(ptr)))
			hasBackupText = true
			procGlobalUnlock.Call(handle)
		}
	}

	procEmptyClipboard.Call()

	// Create a dummy DIB (40 bytes BITMAPINFOHEADER + 4 bytes pixels)
	dibSize := 40 + 4
	hMem, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(dibSize))
	if hMem == 0 {
		procCloseClipboard.Call()
		t.Fatal("GlobalAlloc failed")
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		procCloseClipboard.Call()
		t.Fatal("GlobalLock failed")
	}

	// Write BITMAPINFOHEADER
	header := (*struct {
		biSize          uint32
		biWidth         int32
		biHeight        int32
		biPlanes        uint16
		biBitCount      uint16
		biCompression   uint32
		biSizeImage     uint32
		biXPelsPerMeter int32
		biYPelsPerMeter int32
		biClrUsed       uint32
		biClrImportant  uint32
	})(unsafe.Pointer(ptr))

	header.biSize = 40
	header.biWidth = 1
	header.biHeight = 1
	header.biPlanes = 1
	header.biBitCount = 32
	header.biCompression = 0 // BI_RGB

	// Write 4 pixel bytes (RGBA)
	pixels := (*[4]byte)(unsafe.Pointer(ptr + 40))
	pixels[0] = 0   // B
	pixels[1] = 0   // G
	pixels[2] = 255 // R
	pixels[3] = 255 // A

	procGlobalUnlock.Call(hMem)

	// Set clipboard data
	retSet, _, _ := procSetClipboardData.Call(cfDIB, hMem)
	procCloseClipboard.Call()

	if retSet == 0 {
		t.Fatal("SetClipboardData failed")
	}

	// Restore backup at the end of the test
	defer func() {
		ret, _, _ := procOpenClipboard.Call(0)
		if ret != 0 {
			procEmptyClipboard.Call()
			if hasBackupText {
				writeSystemClipboard(backupText)
			}
			procCloseClipboard.Call()
		}
	}()

	// Now call readSystemClipboard()
	path, ok := readSystemClipboard(nil)
	if !ok {
		t.Fatal("expected readSystemClipboard to return ok=true for image")
	}

	if !strings.Contains(path, "tuios_clipboard_image_") || !strings.HasSuffix(path, ".png") {
		t.Errorf("expected path to be a temporary png file, got: %s", path)
	}

	// Verify the file exists and is not empty
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected temp file to exist: %v", err)
	}
	if info.Size() == 0 {
		t.Error("expected temp file to be non-empty")
	}

	// Cleanup the temp file
	_ = os.Remove(path)
}
