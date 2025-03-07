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

package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/websocket"

	set "github.com/deckarep/golang-set"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/p2p/netutil"
)

// websocketJSONCodec is a custom JSON codec with payload size enforcement and
// special number parsing.
var websocketJSONCodec = websocket.Codec{
	// Marshal is the stock JSON marshaller used by the websocket library too.
	Marshal: func(v interface{}) ([]byte, byte, error) {
		msg, err := json.Marshal(v)
		return msg, websocket.TextFrame, err
	},
	// Unmarshal is a specialized unmarshaller to properly convert numbers.
	Unmarshal: func(msg []byte, payloadType byte, v interface{}) error {
		dec := json.NewDecoder(bytes.NewReader(msg))
		dec.UseNumber()
		return dec.Decode(v)
	},
}

// WebsocketHandler returns a handler that serves JSON-RPC to WebSocket connections.
//
// allowedOrigins should be a comma-separated list of allowed origin URLs.
// To allow connections with any origin, pass "*".
func (srv *Server) WebsocketHandler(allowedOrigins []string, allowedIP netutil.Netlist, reverseproxy bool) http.Handler {
	return websocket.Server{
		Handshake: wsHandshakeValidator(allowedOrigins, allowedIP, reverseproxy),
		Handler:   srv.websocketHandler,
	}
}

func (srv *Server) websocketHandler(conn *websocket.Conn) {
	if conn == nil {
		log.Error("websocket: conn is nil")
		return
	}
	if remote := conn.RemoteAddr(); remote == nil {
		log.Error("websocket: remote address is nil")
		return
	}
	// Create a custom encode/decode pair to enforce payload size and number encoding
	conn.MaxPayloadBytes = maxHTTPRequestContentLength

	encoder := func(v interface{}) error {
		return websocketJSONCodec.Send(conn, v)
	}
	decoder := func(v interface{}) error {
		return websocketJSONCodec.Receive(conn, v)
	}
	name := "websocket"
	log.Warn("websocket: connection", "remote", fmt.Sprintf("%#v", conn))
	// name := fmt.Sprintf("ws:%s", conn.RemoteAddr().String())
	srv.ServeCodec(name, NewCodec(conn, encoder, decoder), OptionMethodInvocation|OptionSubscriptions)
}

// NewWSServer creates a new websocket RPC server around an API provider.
//
// Deprecated: use Server.WebsocketHandler
func NewWSServer(allowedOrigins []string, allowedIP netutil.Netlist, reverseproxy bool, srv *Server) *http.Server {
	return &http.Server{Handler: srv.WebsocketHandler(allowedOrigins, allowedIP, reverseproxy)}
}

// wsHandshakeValidator returns a handler that verifies the origin during the
// websocket upgrade process. When a '*' is specified as an allowed origins all
// connections are accepted.
func wsHandshakeValidator(allowedOrigins []string, allowIPset netutil.Netlist, reverseProxy bool) func(*websocket.Config, *http.Request) error {
	origins := set.NewSet()
	//replacer := strings.NewReplacer(" ", "", "\n", "", "\t", "")

	allowAllOrigins := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
		}
		if origin != "" {
			origins.Add(strings.ToLower(origin))
		}
	}

	// browser/cors: allow only localhost if no allowedOrigins are specified.
	if len(origins.ToSlice()) == 0 {
		log.Warn("websocket: no '--wsorigins' specified, use --wsorigins='*' if you want browser access (CORS)")
		origins.Add("http://localhost")
	}

	log.Info(fmt.Sprintf("Allowed origin(s) for WS RPC interface %v\n", origins.ToSlice()))
	if len(allowIPset) == 1 && allowIPset[0].String() == "0.0.0.0/0" {
		log.Warn("WARNING: WS RPC interface is open to all IPs")
		time.Sleep(time.Second)
	} else {
		log.Info(fmt.Sprintf("Allowed IP(s) for WS RPC interface %s\n", allowIPset.String()))
	}

	f := func(cfg *websocket.Config, req *http.Request) error {
		checkip := func(r *http.Request, reverseProxy bool) error {
			ip := getIP(r, reverseProxy)
			if allowIPset.Contains(ip) {
				return nil
			}
			log.Warn("unwarranted websocket request", "ip", ip)
			return fmt.Errorf("ip not allowed")
		}

		// check ip
		if err := checkip(req, reverseProxy); err != nil {
			return err
		}

		// check origin header
		origin := strings.ToLower(req.Header.Get("Origin"))
		if allowAllOrigins || origins.Contains(origin) {
			return nil
		}
		// log.Warn(fmt.Sprintf("origin '%s' not allowed on WS-RPC interface\n", origin))
		return fmt.Errorf("origin %s not allowed", origin)
	}

	return f
}

var ErrIP = fmt.Errorf("ip not allowed")
var ErrOrigin = fmt.Errorf("origin not allowed")
