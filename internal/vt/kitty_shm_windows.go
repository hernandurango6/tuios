//go:build windows

package vt

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32          = windows.NewLazyDLL("kernel32.dll")
	procOpenFileMappingW = modKernel32.NewProc("OpenFileMappingW")
)

func openFileMapping(desiredAccess uint32, inheritHandle bool, name *uint16) (windows.Handle, error) {
	inherit := uintptr(0)
	if inheritHandle {
		inherit = 1
	}
	r0, _, e1 := procOpenFileMappingW.Call(
		uintptr(desiredAccess),
		inherit,
		uintptr(unsafe.Pointer(name)),
	)
	if r0 == 0 {
		return 0, e1
	}
	return windows.Handle(r0), nil
}

func loadSharedMemory(name string, size int) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("empty shared memory name")
	}

	handle, mappedSize, err := openSharedMemoryMapping(name, size)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)

	addr, err := windows.MapViewOfFile(handle, windows.FILE_MAP_READ, 0, 0, uintptr(mappedSize))
	if err != nil {
		return nil, fmt.Errorf("map view of file: %w", err)
	}
	defer func() { _ = windows.UnmapViewOfFile(addr) }()

	data := unsafe.Slice((*byte)(unsafe.Pointer(addr)), mappedSize)
	result := make([]byte, mappedSize)
	copy(result, data)
	return result, nil
}

func openSharedMemoryMapping(name string, size int) (windows.Handle, int, error) {
	candidates := windowsSharedMemoryNames(name)
	var lastErr error

	for _, candidate := range candidates {
		namePtr, err := windows.UTF16PtrFromString(candidate)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid shared memory name: %w", err)
		}

		handle, err := openFileMapping(windows.FILE_MAP_READ, false, namePtr)
		if err != nil {
			lastErr = err
			continue
		}

		mappedSize := size
		if mappedSize <= 0 {
			var info windows.MemoryBasicInformation
			// Map a single page to query the backing region size.
			probe, err := windows.MapViewOfFile(handle, windows.FILE_MAP_READ, 0, 0, 4096)
			if err != nil {
				windows.CloseHandle(handle)
				lastErr = err
				continue
			}
			if err := windows.VirtualQuery(probe, &info, unsafe.Sizeof(info)); err != nil {
				_ = windows.UnmapViewOfFile(probe)
				windows.CloseHandle(handle)
				lastErr = err
				continue
			}
			_ = windows.UnmapViewOfFile(probe)
			mappedSize = int(info.RegionSize)
		}

		if mappedSize <= 0 {
			windows.CloseHandle(handle)
			return 0, 0, fmt.Errorf("invalid shm size")
		}

		return handle, mappedSize, nil
	}

	if lastErr != nil {
		return 0, 0, fmt.Errorf("open file mapping: %w", lastErr)
	}
	return 0, 0, fmt.Errorf("open file mapping: shared memory %q not found", name)
}

func windowsSharedMemoryNames(name string) []string {
	trimmed := strings.TrimPrefix(name, "/")
	trimmed = strings.TrimPrefix(trimmed, `\`)
	if trimmed == "" {
		return nil
	}

	seen := make(map[string]struct{}, 4)
	add := func(candidate string) []string {
		if candidate == "" {
			return nil
		}
		if _, ok := seen[candidate]; ok {
			return nil
		}
		seen[candidate] = struct{}{}
		return []string{candidate}
	}

	var names []string
	names = append(names, add(trimmed)...)
	names = append(names, add(name)...)
	names = append(names, add(`Local\`+trimmed)...)
	names = append(names, add(`Global\`+trimmed)...)
	return names
}
