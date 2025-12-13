package ipc

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	fileMapAllAccess = 0xF001F
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procOpenFileMapping = kernel32.NewProc("OpenFileMappingA")
	procMapViewOfFile   = kernel32.NewProc("MapViewOfFile")
	procUnmapViewOfFile = kernel32.NewProc("UnmapViewOfFile")
	procCloseHandle     = kernel32.NewProc("CloseHandle")
)

func mapSharedMemory() (uintptr, syscall.Handle, error) {
	namePtr, err := syscall.BytePtrFromString(ShmName)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid shm name string: %w", err)
	}

	// equals to HANDLE OpenFileMappingA(DWORD dwDesiredAccess, BOOL bInheritHandle, LPCSTR lpName);
	hMapFile, _, err := procOpenFileMapping.Call(
		uintptr(fileMapAllAccess),
		0,
		uintptr(unsafe.Pointer(namePtr)),
	)

	if hMapFile == 0 {
		return 0, 0, fmt.Errorf("failed to open shared memory '%s' (is craft.exe running?): %v", ShmName, err)
	}

	// equals to LPVOID MapViewOfFile(HANDLE hFileMappingObject, DWORD dwDesiredAccess, DWORD dwFileOffsetHigh, DWORD dwFileOffsetLow, SIZE_T dwNumberOfBytesToMap);
	addr, _, err := procMapViewOfFile.Call(
		hMapFile,
		uintptr(fileMapAllAccess),
		0,
		0,
		0, // 0 maps the entire file
	)

	if addr == 0 {
		procCloseHandle.Call(hMapFile)
		return 0, 0, fmt.Errorf("failed to map view of file: %v", err)
	}

	return addr, syscall.Handle(hMapFile), nil
}
func unmapSharedMemory(addr uintptr, handle syscall.Handle) {
	if addr != 0 {
		procUnmapViewOfFile.Call(addr)
	}
	if handle != 0 {
		procCloseHandle.Call(uintptr(handle))
	}
}
