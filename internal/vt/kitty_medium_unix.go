//go:build unix

package vt

func sharedMemoryFSPath(name string) string {
	return "/dev/shm/" + name
}
