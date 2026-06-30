package vt

import "fmt"

// ReadKittyMediumData reads pixel data for file-based kitty transmission modes.
func ReadKittyMediumData(cmd *KittyCommand) ([]byte, error) {
	if cmd == nil {
		return nil, fmt.Errorf("nil kitty command")
	}

	switch cmd.Medium {
	case KittyMediumSharedMemory:
		return loadSharedMemory(cmd.FilePath, cmd.Size)
	case KittyMediumFile, KittyMediumTempFile:
		return LoadFileData(cmd.FilePath)
	default:
		return nil, fmt.Errorf("unsupported kitty medium: %c", cmd.Medium)
	}
}

// SharedMemoryFSPath returns the POSIX filesystem path for a kitty shared-memory
// object name. The second return value is false on platforms that do not expose
// shared memory as a regular file path.
func SharedMemoryFSPath(name string) (string, bool) {
	if name == "" {
		return "", false
	}
	path := sharedMemoryFSPath(name)
	if path == "" {
		return "", false
	}
	return path, true
}
