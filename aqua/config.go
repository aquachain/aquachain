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

package aqua

import (
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"gitlab.com/aquachain/aquachain/aqua/downloader"
	"gitlab.com/aquachain/aquachain/aqua/gasprice"
	"gitlab.com/aquachain/aquachain/common/config"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/core"
)

type Config = config.Aquaconfig // TODO remove

// DefaultConfig contains default settings for use on the Aquachain main net.
var DefaultConfig = NewDefaultConfig()

func NewDefaultConfig() *Config {
	return &config.Aquaconfig{
		SyncMode: downloader.FullSync,
		Aquahash: &aquahash.Config{
			CacheDir:       "aquahash",
			CachesInMem:    1,
			CachesOnDisk:   0,
			DatasetsInMem:  0,
			DatasetsOnDisk: 0,
			DatasetDir:     DefaultDatasetDirByOS(),
			PowMode:        aquahash.ModeNormal,
			StartVersion:   0,
		},
		ChainId:       61717561,
		DatabaseCache: 768,
		TrieCache:     256,
		TrieTimeout:   5 * time.Minute,
		GasPrice:      1_000_000_000, // 1.00 gwei
		NoPruning:     true,
		TxPool:        core.DefaultTxPoolConfig,
		GPO: gasprice.Config{
			Blocks:     20,
			Percentile: 60,
		},
	}
}

func DefaultDatasetDirByOS() string {
	if custom := sense.Getenv("AQUAHASH_DATASET_DIR"); custom != "" {
		return custom
	}
	home := sense.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Aquahash")
	}
	return filepath.Join(home, ".aquahash")
}
