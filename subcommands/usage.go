// Copyright 2018 The aquachain Authors
// This file is part of aquachain.
//
// aquachain is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// aquachain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with aquachain. If not, see <http://www.gnu.org/licenses/>.

// Contains the aquachain command usage template and generator.

package subcommands

import (
	"io"
	"sort"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/internal/debug"
	"gitlab.com/aquachain/aquachain/subcommands/aquaflags"
)

const logo = `                              _           _
  __ _  __ _ _   _  __ _  ___| |__   __ _(_)_ __
 / _ '|/ _' | | | |/ _' |/ __| '_ \ / _' | | '_ \
| (_| | (_| | |_| | (_| | (__| | | | (_| | | | | |
 \__,_|\__, |\__,_|\__,_|\___|_| |_|\__,_|_|_| |_|
          |_|` + "Update Often! https://gitlab.com/aquachain/aquachain\n\n"

// AppHelpTemplate is the test template for the default, global app help topic.
var AppHelpTemplate = logo + `NAME:
   {{.App.Name}} - {{.App.Usage}}

    Copyright 2018-2025 The aquachain authors
    Copyright 2013-2018 The go-ethereum authors

USAGE:
   {{.App.Name}} [options]{{if .App.Commands}} command [command options]{{end}} {{if .App.ArgsUsage}}{{.App.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if .App.Version}}
VERSION:
   {{.App.Version}}
   {{end}}{{if len .App.Authors}}
AUTHOR(S):
   {{range .App.Authors}}{{ . }}{{end}}
   {{end}}{{if .App.Commands}}
COMMANDS:
   {{range .App.Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
   {{end}}{{end}}{{if .FlagGroups}}
{{range .FlagGroups}}{{.Name}} OPTIONS:
  {{range .Flags}}{{.}}
  {{end}}
{{end}}{{end}}{{if .App.Copyright }}
COPYRIGHT:
   {{.App.Copyright}}
   {{end}}
`

// flagGroup is a collection of flags belonging to a single topic.
type flagGroup struct {
	Name  string
	Flags []cli.Flag
}

// AppHelpFlagGroups is the application flags, grouped by functionality.
var AppHelpFlagGroups = []flagGroup{
	{
		Name: "Aquachain",
		Flags: []cli.Flag{
			aquaflags.ConfigFileFlag,
			aquaflags.DataDirFlag,
			aquaflags.KeyStoreDirFlag,
			aquaflags.UseUSBFlag,

			aquaflags.SyncModeFlag,
			aquaflags.ChainFlag,
			aquaflags.GCModeFlag,
			aquaflags.AquaStatsURLFlag,
			aquaflags.IdentityFlag,
			aquaflags.HF8MainnetFlag,
			aquaflags.AlertModeFlag,
			aquaflags.DoitNowFlag,
			aquaflags.NoKeysFlag,
			aquaflags.RPCBehindProxyFlag,
		},
	},
	{Name: "DEVELOPER CHAIN",
		Flags: []cli.Flag{
			aquaflags.DeveloperFlag,
			aquaflags.DeveloperPeriodFlag,
		},
	},
	{
		Name: "AQUAHASH",
		Flags: []cli.Flag{
			aquaflags.AquahashCacheDirFlag,
			aquaflags.AquahashCachesInMemoryFlag,
			aquaflags.AquahashCachesOnDiskFlag,
			aquaflags.AquahashDatasetDirFlag,
			aquaflags.AquahashDatasetsInMemoryFlag,
			aquaflags.AquahashDatasetsOnDiskFlag,
		},
	},
	{
		Name: "TRANSACTION POOL",
		Flags: []cli.Flag{
			aquaflags.TxPoolNoLocalsFlag,
			aquaflags.TxPoolJournalFlag,
			aquaflags.TxPoolRejournalFlag,
			aquaflags.TxPoolPriceLimitFlag,
			aquaflags.TxPoolPriceBumpFlag,
			aquaflags.TxPoolAccountSlotsFlag,
			aquaflags.TxPoolGlobalSlotsFlag,
			aquaflags.TxPoolAccountQueueFlag,
			aquaflags.TxPoolGlobalQueueFlag,
			aquaflags.TxPoolLifetimeFlag,
		},
	},
	{
		Name: "PERFORMANCE TUNING",
		Flags: []cli.Flag{
			aquaflags.CacheFlag,
			aquaflags.CacheDatabaseFlag,
			aquaflags.CacheGCFlag,
			aquaflags.TrieCacheGenFlag,
		},
	},
	{
		Name: "ACCOUNT",
		Flags: []cli.Flag{
			aquaflags.UnlockedAccountFlag,
			aquaflags.PasswordFileFlag,
		},
	},
	{
		Name: "API AND CONSOLE",
		Flags: []cli.Flag{
			aquaflags.RPCEnabledFlag,
			aquaflags.RPCListenAddrFlag,
			aquaflags.RPCPortFlag,
			aquaflags.RPCApiFlag,
			aquaflags.WSEnabledFlag,
			aquaflags.WSListenAddrFlag,
			aquaflags.WSPortFlag,
			aquaflags.WSApiFlag,
			aquaflags.WSAllowedOriginsFlag,
			aquaflags.IPCDisabledFlag,
			aquaflags.IPCPathFlag,
			aquaflags.RPCCORSDomainFlag,
			aquaflags.RPCVirtualHostsFlag,
			aquaflags.JavascriptDirectoryFlag,
			aquaflags.ExecFlag,
			aquaflags.PreloadJSFlag,
		},
	},
	{
		Name: "NETWORKING",
		Flags: []cli.Flag{
			aquaflags.BootnodesFlag,
			aquaflags.ListenPortFlag,
			aquaflags.MaxPeersFlag,
			aquaflags.MaxPendingPeersFlag,
			aquaflags.NATFlag,
			aquaflags.NoDiscoverFlag,
			aquaflags.OfflineFlag,
			aquaflags.NetrestrictFlag,
			aquaflags.NodeKeyFileFlag,
			aquaflags.NodeKeyHexFlag,
		},
	},
	{
		Name: "MINER",
		Flags: []cli.Flag{
			aquaflags.MiningEnabledFlag,
			aquaflags.MinerThreadsFlag,
			aquaflags.AquabaseFlag,
			aquaflags.TargetGasLimitFlag,
			aquaflags.GasPriceFlag,
			aquaflags.ExtraDataFlag,
		},
	},
	{
		Name: "GAS PRICE ORACLE",
		Flags: []cli.Flag{
			aquaflags.GpoBlocksFlag,
			aquaflags.GpoPercentileFlag,
		},
	},
	{
		Name: "VIRTUAL MACHINE",
		Flags: []cli.Flag{
			aquaflags.VMEnableDebugFlag,
		},
	},
	{
		Name: "LOGGING AND DEBUGGING",
		Flags: append([]cli.Flag{
			aquaflags.MetricsEnabledFlag,
			aquaflags.FakePoWFlag,
			aquaflags.NoCompactionFlag,
		}, debug.Flags...),
	},

	{
		Name: "DEPRECATED",
	},
	{
		Name:  "MISC",
		Flags: []cli.Flag{aquaflags.FastSyncFlag},
	},
}

// byCategory sorts an array of flagGroup by Name in the order
// defined in AppHelpFlagGroups.
type byCategory []flagGroup

func (a byCategory) Len() int      { return len(a) }
func (a byCategory) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byCategory) Less(i, j int) bool {
	iCat, jCat := a[i].Name, a[j].Name
	iIdx, jIdx := len(AppHelpFlagGroups), len(AppHelpFlagGroups) // ensure non categorized flags come last

	for i, group := range AppHelpFlagGroups {
		if iCat == group.Name {
			iIdx = i
		}
		if jCat == group.Name {
			jIdx = i
		}
	}

	return iIdx < jIdx
}

func compareNames(a1, a2 []string) bool {
	if len(a1) != len(a2) {
		return false
	}
	for i, v := range a1 {
		if v != a2[i] {
			return false
		}
	}
	return true
}

func flagCategory(flag cli.Flag) string {
	for _, category := range AppHelpFlagGroups {
		for _, flg := range category.Flags {
			if compareNames(flg.Names(), flag.Names()) {
				return category.Name
			}
		}
	}
	return "MISC"
}

func InitHelp() {
	// Override the default app help template
	cli.RootCommandHelpTemplate = AppHelpTemplate

	// Define a one shot struct to pass to the usage template
	type helpData struct {
		App        interface{}
		FlagGroups []flagGroup
	}

	// Override the default app help printer, but only for the global app help
	originalHelpPrinter := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, tmpl string, data interface{}) {
		if tmpl == AppHelpTemplate {
			// Iterate over all the flags and add any uncategorized ones
			categorized := make(map[string]struct{})
			for _, group := range AppHelpFlagGroups {
				for _, flag := range group.Flags {
					categorized[flag.String()] = struct{}{}
				}
			}
			cmd, ok := data.(*cli.Command)
			if !ok {
				log.Warn("unexpected data type for app help template", "type", data)
				originalHelpPrinter(w, tmpl, data)
				return
			}
			flags := cmd.Flags
			uncategorized := []cli.Flag{}
			for _, flag := range flags {
				if _, ok := categorized[flag.String()]; !ok {
					uncategorized = append(uncategorized, flag)
				}
			}
			if len(uncategorized) > 0 {
				// Append all ungategorized options to the misc group
				miscs := len(AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags)
				AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags = append(AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags, uncategorized...)

				// Make sure they are removed afterwards
				defer func() {
					AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags = AppHelpFlagGroups[len(AppHelpFlagGroups)-1].Flags[:miscs]
				}()
			}
			// Render out custom usage screen
			originalHelpPrinter(w, tmpl, helpData{data, AppHelpFlagGroups})
		} else if tmpl == CommandHelpTemplate {
			// Iterate over all command specific flags and categorize them
			categorized := make(map[string][]cli.Flag)
			for _, flag := range data.(cli.Command).Flags {
				if _, ok := categorized[flag.String()]; !ok {
					categorized[flagCategory(flag)] = append(categorized[flagCategory(flag)], flag)
				}
			}

			// sort to get a stable ordering
			sorted := make([]flagGroup, 0, len(categorized))
			for cat, flgs := range categorized {
				sorted = append(sorted, flagGroup{cat, flgs})
			}
			sort.Sort(byCategory(sorted))

			// add sorted array to data and render with default printer
			originalHelpPrinter(w, tmpl, map[string]interface{}{
				"cmd":              data,
				"categorizedFlags": sorted,
			})
		} else {
			originalHelpPrinter(w, tmpl, data)
		}
	}
}
