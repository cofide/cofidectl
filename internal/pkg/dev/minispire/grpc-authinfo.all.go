//go:build !linux

package minispire

func getProcessInfo(fd uintptr) (int32, uint32, uint32) {
	// This function is not implemented for this platform
	return 0, 0, 0
}
