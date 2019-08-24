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

package main

import (
	"context"
	"fmt"
	"log"

	"gitlab.com/aquachain/aquachain/opt/aquaclient"
	rpc "gitlab.com/aquachain/aquachain/rpc/rpcclient"

	"github.com/btcsuite/btcd/rpcclient"
)

func main() {
	// Connect to local bitcoin core RPC server using HTTP POST mode.
	connCfg := &rpcclient.ConnConfig{
		Host: "localhost:8543",
		//		User:         "yourrpcuser",
		//		Pass:         "yourrpcpass",
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}
	rpcc, err := rpc.DialHTTP("http://localhost:8543")
	if err != nil {
		log.Fatal(err)
	}
	aquaclient1 := aquaclient.NewClient(rpcc)
	latest, err := aquaclient1.BlockByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("AQUA:", latest.Number())

	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()

	// Get the current block count.
	blockCount, err := client.GetBlockCount()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("BTC: %d", blockCount)
}
