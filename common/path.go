// Copyright 2018 The aquachain Authors
// This file is part of the aquachain library.
//
// The aquachain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The aquachain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the aquachain library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func shorten(s string, n int) string {
	l := len(s)
	if l < n {
		return s
	}
	return s[:n]
}

func ShortGoVersion() string {
	runtimeVersion := runtime.Version()
	// example output: devel go1.20-cc1b20e8ad Sat Sep 17 02:56:51 2022 +0000
	if strings.Contains(runtimeVersion, "devel ") {
		runtimeVersion = strings.TrimPrefix(runtimeVersion, "devel ")

		runtimeVersion = strings.Replace(runtimeVersion, "go", "godev", -1)
	}
	return shorten(strings.Split(runtimeVersion, "-")[0], 10) // go version
}

// MakeName creates a node name that follows the aquachain convention
// for such names. It adds the operation system name and Go runtime version
// the name.
func MakeName(name, version string) string {
	return fmt.Sprintf("%s/v%s/%s/%s", name, version, runtime.GOOS, ShortGoVersion())
}

func FileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return true
}

func AbsolutePath(Datadir string, filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(Datadir, filename)
}

// EnvBool returns false if empty/unset/falsy, true otherwise
func EnvBool(name string) bool {
	x, ok := os.LookupEnv(name)
	if !ok {
		return false
	}
	if x == "" {
		return false
	}
	if x == "0" {
		return false
	}
	if x == "1" {
		return true
	}
	x = strings.TrimSpace(x)
	x = strings.ToLower(x)
	if x == "false" || x == "no" || x == "off" {
		return false
	}
	return true
}
