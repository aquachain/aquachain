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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net"
	"net/http"
	"time"

	"strings"

	"github.com/rs/cors"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/p2p/netutil"
)

const (
	contentType                 = "application/json"
	maxHTTPRequestContentLength = 1024 * 128
)

// httpReadWriteNopCloser wraps a io.Reader and io.Writer with a NOP Close method.
type httpReadWriteNopCloser struct {
	io.Reader
	io.Writer
}

// Close does nothing and returns always nil
func (t *httpReadWriteNopCloser) Close() error {
	return nil
}

// NewHTTPServer creates a new HTTP RPC server around an API provider.
//
// Deprecated: Server implements http.Handler
func NewHTTPServer(cors []string, vhosts []string, allowIP netutil.Netlist, behindreverseproxy bool, srv *Server) *http.Server {
	// Check IPs, hostname, then CORS (in that order)
	handler := newAllowIPHandler(allowIP, behindreverseproxy, newVHostHandler(vhosts, newCorsHandler(newLoggedHandler(srv), cors)))
	return &http.Server{Handler: handler, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second, IdleTimeout: time.Second * 30}
}

// ServeHTTP serves JSON-RPC requests over HTTP.
func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Permit dumb empty requests for remote health-checks (AWS)
	if r.Method == http.MethodGet && r.ContentLength == 0 && r.URL.RawQuery == "" {
		return
	}
	if code, err := validateRequest(r); err != nil {
		uip := getIP(r, srv.reverseproxy)
		log.Debug("invalid request", "from", uip, "size", r.ContentLength)
		enc := json.NewEncoder(w)
		enc.SetIndent(" ", " ")
		w.WriteHeader(code)
		enc.Encode(map[string]string{"error": err.Error()})
		return
	}
	// All checks passed, create a codec that reads direct from the request body
	// untilEOF and writes the response to w and order the server to process a
	// single request.
	body := io.LimitReader(r.Body, maxHTTPRequestContentLength)
	codec := NewJSONCodec(&httpReadWriteNopCloser{body, w})
	defer codec.Close()

	w.Header().Set("content-type", contentType)
	srv.ServeSingleRequest(codec, OptionMethodInvocation)
}

// validateRequest returns a non-zero response code and error message if the
// request is invalid.
func validateRequest(r *http.Request) (int, error) {
	if r.Method == http.MethodPut || r.Method == http.MethodDelete {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}
	if r.ContentLength > maxHTTPRequestContentLength {
		err := fmt.Errorf("content length too large (%d>%d)", r.ContentLength, maxHTTPRequestContentLength)
		return http.StatusRequestEntityTooLarge, err
	}
	mt, _, err := mime.ParseMediaType(r.Header.Get("content-type"))
	if r.Method != http.MethodOptions && (err != nil || mt != contentType) {
		err := fmt.Errorf("invalid content type, only %s is supported", contentType)
		return http.StatusUnsupportedMediaType, err
	}
	return 0, nil
}

func newLoggedHandler(srv *Server) http.Handler {
	return loggedHandler{srv, srv.reverseproxy} // TODO: set reverseproxy!
}

func (l loggedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqid := fmt.Sprintf("%02X", rand.Uint32())
	lrw := &lrwriter{ResponseWriter: w, statusCode: http.StatusOK}
	logfn := log.Debug
	uip := getIP(r, l.reverseproxy)
	logfn("<<< http-rpc: "+reqid, "from", uip, "path", r.URL.Path, "ua", r.UserAgent(), "method", r.Method, "host", r.Host, "size", r.ContentLength)

	l.h.ServeHTTP(lrw, r)
	if lrw.statusCode != 200 {
		logfn = log.Warn
	}
	logfn(">>> http-rpc: "+reqid, "code", lrw.statusCode, "status", http.StatusText(lrw.statusCode))
}

// override WriteHeader, just saving response code
func (lrw *lrwriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

type loggedHandler struct {
	h            http.Handler
	reverseproxy bool
}
type lrwriter struct {
	http.ResponseWriter
	statusCode int
}

func newCorsHandler(h http.Handler, allowedOrigins []string) http.Handler {
	// disable CORS support if user has not specified a custom CORS configuration
	if len(allowedOrigins) == 0 {
		return h
	}
	c := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{http.MethodPost, http.MethodGet},
		MaxAge:         600,
		AllowedHeaders: []string{"*"},
	})
	return c.Handler(h)
}

// virtualHostHandler is a handler which validates the Host-header of incoming requests.
// The virtualHostHandler can prevent DNS rebinding attacks, which do not utilize CORS-headers,
// since they do in-domain requests against the RPC api. Instead, we can see on the Host-header
// which domain was used, and validate that against a whitelist.
type virtualHostHandler struct {
	vhosts map[string]struct{}
	next   http.Handler
}

// ServeHTTP serves JSON-RPC requests over HTTP, implements http.Handler
func (h *virtualHostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// if r.Host is not set, we can continue serving since a browser would set the Host header
	if r.Host == "" {
		h.next.ServeHTTP(w, r)
		return
	}
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		// Either invalid (too many colons) or no port specified
		host = r.Host
	}
	if ipAddr := net.ParseIP(host); ipAddr != nil {
		// It's an IP address, we can serve that
		h.next.ServeHTTP(w, r)
		return

	}
	// Not an ip address, but a hostname. Need to validate
	if _, exist := h.vhosts["*"]; exist {
		h.next.ServeHTTP(w, r)
		return
	}
	if _, exist := h.vhosts[host]; exist {
		h.next.ServeHTTP(w, r)
		return
	}
	http.Error(w, "invalid host specified", http.StatusForbidden)
}

func newVHostHandler(vhosts []string, next http.Handler) http.Handler {
	vhostMap := make(map[string]struct{})
	for _, allowedHost := range vhosts {
		vhostMap[strings.ToLower(allowedHost)] = struct{}{}
	}
	return &virtualHostHandler{vhostMap, next}
}

// allowIPHandler is a handler which only allows certain IP
type allowIPHandler struct {
	allowedIPs   netutil.Netlist
	next         http.Handler
	reverseproxy bool // if behind a reverse proxy (uses X-FORWARDED-FOR header)
}

func getIP(r *http.Request, reverseproxy bool) net.IP {
	if reverseproxy {
		for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
			addresses := strings.Split(r.Header.Get(h), ",")
			// march from right to left until we get a public address
			// that will be the address right before our proxy.
			for i := len(addresses) - 1; i >= 0; i-- {
				// header can contain spaces too, strip those out.
				ip := strings.TrimSpace(addresses[i])
				realIP := net.ParseIP(ip)
				if realIP == nil {
					continue
				}
				if !realIP.IsGlobalUnicast() || netutil.IsLAN(realIP) || netutil.IsSpecialNetwork(realIP) {
					// bad address, go to next
					continue
				}

				return net.ParseIP(ip)
			}
		}
		// no X-Forwarded-For header ...
	}
	remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Either invalid (too many colons) or no port specified
		remoteAddr = strings.Split(r.RemoteAddr, ":")[0]
	}
	return net.ParseIP(remoteAddr)
}

// ServeHTTP serves JSON-RPC requests over HTTP, implements http.Handler
func (h *allowIPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip := getIP(r, h.reverseproxy)
	log.Trace("allowip: checking vs allow IPs", "ip", ip)
	if h.allowedIPs.Contains(ip) {
		h.next.ServeHTTP(w, r)
		return
	}
	log.Warn("allowip: blocking http rpc connection", "OffendingIP", ip, "User-Agent", r.UserAgent())
	http.Error(w, "", http.StatusForbidden)
}

func newAllowIPHandler(allowIPMap netutil.Netlist, behindreverseproxy bool, next http.Handler) http.Handler {
	return &allowIPHandler{allowIPMap, next, behindreverseproxy}
}
