// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package minispire

import (
	"log"
	"syscall"
)

func getProcessInfo(fd uintptr) (int32, uint32, uint32) {
	cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		log.Printf("unable to get peer credentials: %v", err)
		return 0, 0, 0
	}
	return int32(cred.Pid), uint32(cred.Uid), uint32(cred.Gid)
}
