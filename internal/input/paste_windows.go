//go:build windows

package input

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/Gaurav-Gosain/tuios/internal/app"
)

const (
	cfUnicodeText = 13
	cfBitmap      = 2
	cfDIB         = 8
)

const gmemMoveable = 0x0002

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procOpenClipboard              = modUser32.NewProc("OpenClipboard")
	procCloseClipboard             = modUser32.NewProc("CloseClipboard")
	procEmptyClipboard             = modUser32.NewProc("EmptyClipboard")
	procGetClipboardData           = modUser32.NewProc("GetClipboardData")
	procSetClipboardData           = modUser32.NewProc("SetClipboardData")
	procIsClipboardFormatAvailable = modUser32.NewProc("IsClipboardFormatAvailable")
	procGlobalAlloc                = modKernel32.NewProc("GlobalAlloc")
	procGlobalLock                 = modKernel32.NewProc("GlobalLock")
	procGlobalUnlock               = modKernel32.NewProc("GlobalUnlock")
)

// readSystemClipboard reads UTF-16 text or an image from the Windows clipboard.
// If the clipboard holds an image (CF_DIB/CF_BITMAP) the image is saved to a
// temporary PNG file and its path is returned so the caller can paste the path.
func readSystemClipboard(o *app.OS) (string, bool) {
	// --- Step 1: try text ---
	text, hasText := func() (string, bool) {
		ret, _, _ := procOpenClipboard.Call(0)
		if ret == 0 {
			return "", false
		}
		defer procCloseClipboard.Call()

		handle, _, _ := procGetClipboardData.Call(cfUnicodeText)
		if handle == 0 {
			return "", false
		}

		ptr, _, _ := procGlobalLock.Call(handle)
		if ptr == 0 {
			return "", false
		}
		defer procGlobalUnlock.Call(handle)

		return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(ptr))), true
	}()

	if hasText {
		return text, true
	}

	// --- Step 2: try image (IsClipboardFormatAvailable does not require an open clipboard) ---
	retDIB, _, _ := procIsClipboardFormatAvailable.Call(cfDIB)
	retBitmap, _, _ := procIsClipboardFormatAvailable.Call(cfBitmap)
	if retDIB == 0 && retBitmap == 0 {
		return "", false
	}

	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("tuios_clipboard_image_%d.png", time.Now().UnixNano()))

	// Use Windows PowerShell (always available) to decode the clipboard image and
	// save it as PNG. The -STA flag is required for Windows.Forms clipboard access.
	psCmd := fmt.Sprintf(
		`Add-Type -AssemblyName System.Windows.Forms; `+
			`Add-Type -AssemblyName System.Drawing; `+
			`$img = [System.Windows.Forms.Clipboard]::GetImage(); `+
			`if ($img) { $img.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png) }`,
		tempFile,
	)
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-Command", psCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Run(); err != nil {
		if o != nil {
			o.LogError("clipboard image: powershell failed: %v", err)
		}
		return "", false
	}

	info, err := os.Stat(tempFile)
	if err != nil || info.Size() == 0 {
		if o != nil {
			o.LogError("clipboard image: temp file missing or empty: %v", err)
		}
		return "", false
	}

	if o != nil {
		o.LogInfo("clipboard image saved: %s (%d bytes)", tempFile, info.Size())
	}
	return tempFile, true
}

// writeSystemClipboard writes UTF-16 text to the Windows clipboard.
func writeSystemClipboard(text string) bool {
	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return false
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()

	utf16, err := windows.UTF16FromString(text)
	if err != nil || len(utf16) == 0 {
		return false
	}

	size := uintptr(len(utf16) * 2)
	handle, _, _ := procGlobalAlloc.Call(gmemMoveable, size)
	if handle == 0 {
		return false
	}

	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		return false
	}

	dest := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(utf16))
	copy(dest, utf16)
	procGlobalUnlock.Call(handle)

	ret, _, _ = procSetClipboardData.Call(cfUnicodeText, handle)
	return ret != 0
}
