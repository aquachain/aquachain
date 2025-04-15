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

package params

import (
	"encoding/base64"
	"fmt"
)

// regenerate to synchronize /VERSION file with below values
//go:generate bash ./generate_version.bash

const (
	VersionMajor = 1  // Major version component of the current release
	VersionMinor = 7  // Minor version component of the current release
	VersionPatch = 18 // Patch version component of the current release
)

var (
	VersionMeta = "dev" // Version metadata to append to the version string (replaced by Makefile)
)

// Version holds the textual version string.
var Version = func() string {
	v := fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

func VersionWithCommit(gitCommit string) string {
	vsn := Version
	if len(gitCommit) == 0 {
		gitCommit = "???????"
	}
	if len(gitCommit) >= 6 {
		vsn += "-" + gitCommit[:6]
	}
	return vsn
}

// set with -X linker flag
var buildtags string

func BuildTags() string {
	if buildtags == "" {
		return ""
	}
	b, err := base64.RawStdEncoding.DecodeString(buildtags)
	if err != nil {
		panic(err)
	}
	return string(b)
}
