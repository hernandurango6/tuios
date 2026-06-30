//go:build windows

package vt

import (
	"fmt"
	"os"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestLoadSharedMemoryWindows(t *testing.T) {
	name := fmt.Sprintf("tuios-test-shm-%d", os.Getpid())
	payload := []byte("kitty-shm-windows-test-data")
	size := len(payload)

	namePtr, err := windows.UTF16PtrFromString(`Local\` + name)
	if err != nil {
		t.Fatalf("UTF16PtrFromString: %v", err)
	}

	handle, err := windows.CreateFileMapping(
		windows.InvalidHandle,
		nil,
		windows.PAGE_READWRITE,
		0,
		uint32(size),
		namePtr,
	)
	if err != nil {
		t.Fatalf("CreateFileMapping: %v", err)
	}
	defer windows.CloseHandle(handle)

	addr, err := windows.MapViewOfFile(handle, windows.FILE_MAP_WRITE, 0, 0, uintptr(size))
	if err != nil {
		t.Fatalf("MapViewOfFile: %v", err)
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(addr)), size), payload)
	if err := windows.UnmapViewOfFile(addr); err != nil {
		t.Fatalf("UnmapViewOfFile: %v", err)
	}

	got, err := loadSharedMemory(name, size)
	if err != nil {
		t.Fatalf("loadSharedMemory: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("got %q, want %q", got, payload)
	}
}

func TestWindowsSharedMemoryNames(t *testing.T) {
	names := windowsSharedMemoryNames("/mpv-kitty-deadbeef")
	if len(names) < 3 {
		t.Fatalf("expected multiple name candidates, got %v", names)
	}
	if names[0] != "mpv-kitty-deadbeef" {
		t.Fatalf("first candidate = %q, want mpv-kitty-deadbeef", names[0])
	}
}
