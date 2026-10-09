// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build linux || darwin

package update

import (
	"golang.org/x/sys/unix"
	"os"
)

func supportedPlatform() bool  { return true }
func writable(dir string) bool { return unix.Access(dir, unix.W_OK) == nil }
func lockFile(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, err
	}
	return func() { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN); _ = f.Close() }, nil
}
