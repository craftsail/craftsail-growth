// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !linux && !darwin

package update

func supportedPlatform() bool         { return false }
func writable(string) bool            { return false }
func lockFile(string) (func(), error) { return nil, ErrUnsupported }
