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
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"gitlab.com/aquachain/aquachain/aqua/downloader"
	"gitlab.com/aquachain/aquachain/aqua/gasprice"
	"gitlab.com/aquachain/aquachain/common/config"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/core"
)

type Config = config.Aquaconfig // TODO remove

// DefaultConfig contains default settings for use on the Aquachain main net.
var DefaultConfig = &config.Aquaconfig{
	SyncMode: downloader.FullSync,
	Aquahash: aquahash.Config{
		CacheDir:       "aquahash",
		CachesInMem:    1,
		CachesOnDisk:   0,
		DatasetsInMem:  0,
		DatasetsOnDisk: 0,
	},
	ChainId:       61717561,
	DatabaseCache: 768,
	TrieCache:     256,
	TrieTimeout:   5 * time.Minute,
	GasPrice:      big.NewInt(10000000), // 0.01 gwei
	NoPruning:     true,
	TxPool:        core.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     20,
		Percentile: 60,
	},
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "windows" {
		DefaultConfig.Aquahash.DatasetDir = filepath.Join(home, "AppData", "Aquahash")
	} else {
		DefaultConfig.Aquahash.DatasetDir = filepath.Join(home, ".aquahash")
	}
}
