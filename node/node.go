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

package node

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/event"
	"gitlab.com/aquachain/aquachain/aquadb"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/alerts"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/internal/debug"
	"gitlab.com/aquachain/aquachain/internal/flock"
	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/p2p/netutil"
	"gitlab.com/aquachain/aquachain/params"
	"gitlab.com/aquachain/aquachain/rpc"
	rpcclient "gitlab.com/aquachain/aquachain/rpc/rpcclient"
)

// Node is a container on which services can be registered.
type Node struct {
	closemain func()
	ctx       context.Context
	eventmux  *event.TypeMux // Event multiplexer used between the services of a stack
	config    *Config
	accman    *accounts.Manager

	ephemeralKeystore string         // if non-empty, the key directory that will be removed by Stop
	instanceDirLock   flock.Releaser // prevents concurrent use of instance directory

	serverConfig *p2p.Config
	server       *p2p.Server // Currently running P2P networking layer

	serviceFuncs []ServiceConstructor     // Service constructors (in dependency order)
	services     map[reflect.Type]Service // Currently running services

	rpcAPIs       []rpc.API   // List of APIs currently provided by the node
	inprocHandler *rpc.Server // In-process RPC request handler to process the API requests

	ipcEndpoint string       // IPC endpoint to listen at (empty = IPC disabled)
	ipcListener net.Listener // IPC RPC listener socket to serve API requests
	ipcHandler  *rpc.Server  // IPC RPC request handler to process the API requests

	httpEndpoint  string       // HTTP endpoint (interface + port) to listen at (empty = HTTP disabled)
	httpWhitelist []string     // HTTP RPC modules to allow through this endpoint
	httpListener  net.Listener // HTTP RPC listener socket to server API requests
	httpHandler   *rpc.Server  // HTTP RPC request handler to process the API requests

	wsEndpoint string       // Websocket endpoint (interface + port) to listen at (empty = websocket disabled)
	wsListener net.Listener // Websocket RPC listener socket to server API requests
	wsHandler  *rpc.Server  // Websocket RPC request handler to process the API requests

	stop     chan struct{} // Channel to wait for termination notifications
	lock     sync.RWMutex
	chaincfg *params.ChainConfig

	log log.LoggerI
}

func (n *Node) Context() context.Context {
	return n.ctx
}

// New creates a new P2P node, ready for protocol registration.
func New(conf *Config) (*Node, error) {
	if conf.Context == nil {
		panic("Context not set")
	}
	if conf.CloseMain == nil {
		panic("CloseMain not set")
		// return nil, errors.New("CloseMain not set")
	}

	chaincfg := conf.P2P.ChainConfig()
	if chaincfg == nil {
		chaincfg = params.GetChainConfigByChainId(big.NewInt(int64(conf.P2P.ChainId)))
	}
	if chaincfg == nil {
		log.Warn("node: no chain config for chainID", "chainID", conf.P2P.ChainId)
	}
	if conf.Name == "" {
		return nil, errors.New("node.Config.Name not set")
	}

	// Copy config and resolve the datadir so future changes to the current
	// working directory don't affect the node.
	confCopy := *conf
	conf = &confCopy
	if conf.DataDir != "" {
		absdatadir, err := filepath.Abs(conf.DataDir)
		if err != nil {
			return nil, err
		}
		conf.DataDir = absdatadir
	}
	// Ensure that the instance name doesn't cause weird conflicts with
	// other files in the data directory.
	if strings.ContainsAny(conf.Name, `/\`) {
		return nil, errors.New(`Config.Name must not contain '/' or '\'`)
	}
	if conf.Name == datadirDefaultKeyStore {
		return nil, errors.New(`Config.Name cannot be "` + datadirDefaultKeyStore + `"`)
	}
	if strings.HasSuffix(conf.Name, ".ipc") {
		return nil, errors.New(`Config.Name cannot end in ".ipc"`)
	}

	// Ensure that the AccountManager method works before the node has started.
	// We rely on this in cmd/aquachain.
	if conf.KeyStoreDir == "" {
		conf.KeyStoreDir = sense.Getenv("AQUA_KEYSTORE_DIR") // in case of dotenv
	}
	am, ephemeralKeystore, err := makeAccountManager(conf)
	if err != nil {
		return nil, err
	}
	if conf.KeyStoreDir != "" {
		log.Warn("USING KEYSTORE", "custom_dir", conf.KeyStoreDir, "active", fmt.Sprintf("%T", am), "ephemeral", ephemeralKeystore)
	}
	if conf.Logger == nil {
		conf.Logger = log.New()
	}
	log.Info("created a node:", "withKeystore", am != nil, "ephemeralKeystore", ephemeralKeystore)
	// Note: any interaction with Config that would create/touch files
	// in the data directory or instance directory is delayed until Start.
	return &Node{
		ctx:               conf.Context,
		accman:            am,
		ephemeralKeystore: ephemeralKeystore,
		config:            conf,
		serviceFuncs:      []ServiceConstructor{},
		ipcEndpoint:       conf.IPCEndpoint(),
		httpEndpoint:      conf.HTTPEndpoint(),
		wsEndpoint:        conf.WSEndpoint(),
		eventmux:          new(event.TypeMux),
		log:               conf.Logger,
		chaincfg:          chaincfg,
	}, nil
}

// Register injects a new service into the node's stack. The service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (n *Node) Register(constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.server != nil {
		return ErrNodeRunning
	}
	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

var NoCountdown = false

// Start create a live P2P node and starts running it, immediately returning
func (n *Node) Start(ctx context.Context) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.ctx = ctx
	if n.config.P2P.ChainId == 0 {
		return fmt.Errorf("no chain id")
	}
	// Short circuit if the node's already running
	if n.server != nil {
		return ErrNodeRunning
	}
	if err := n.openDataDir(); err != nil {
		return err
	}

	if sense.IsNoKeys() && !n.config.NoKeys {
		n.config.NoKeys = true
		n.log.Warn("NO_KEYS mode enabled (again?)")
	}

	// Initialize the p2p server. This creates the node key and
	// discovery databases.
	n.serverConfig = n.config.P2P
	n.serverConfig.PrivateKey = n.config.NodeKey()
	n.serverConfig.Name = n.config.NodeName()

	if x := strings.Count(n.serverConfig.Name, "/"); x < 2 {
		y := NewDefaultConfig()
		y.Name = "Aquachain"
		shouldbe := GetNodeName(y)
		return fmt.Errorf("node.Node.Name was not created with GetNodeName, should have 3 has %d slashes: %q, should be %q", x, n.serverConfig.Name, shouldbe)
	}
	n.serverConfig.Logger = n.log
	if n.serverConfig.StaticNodes == nil {
		n.serverConfig.StaticNodes = n.config.StaticNodes()
	}
	if n.serverConfig.TrustedNodes == nil {
		n.serverConfig.TrustedNodes = n.config.TrustedNodes()
	}
	if n.serverConfig.NodeDatabase == "" {
		n.serverConfig.NodeDatabase = n.config.NodeDB()
	}
	numTrusted := len(n.serverConfig.TrustedNodes)
	numStatic := len(n.serverConfig.StaticNodes)
	if numTrusted != 0 || numStatic != 0 || !n.serverConfig.NoDiscovery {
		n.log.Info("Static nodes", "static", numStatic, "trusted", numTrusted, "discovery", !n.serverConfig.NoDiscovery)
		for i, s := range n.serverConfig.StaticNodes {
			n.log.Debug("Static node", "index", i, "node", s)
		}
		for i, s := range n.serverConfig.TrustedNodes {
			n.log.Debug("Trusted node", "index", i, "node", s)
		}

	}
	running := &p2p.Server{Config: n.serverConfig}
	n.log.Info("Starting peer-to-peer node", "instance", n.serverConfig.Name, "listening", n.serverConfig.ListenAddr)
	getport := func(s string) string {
		_, port, _ := net.SplitHostPort(s)
		if port == "0" {
			port = "???"
		}
		return port
	}
	if len(n.config.HTTPVirtualHosts) == 0 || len(n.config.HTTPVirtualHosts[0]) == 0 {
		log.Warn("no virtual hosts, using default (any)")
		n.config.HTTPVirtualHosts = []string{"*"}
	}

	// do startup alert if enabled
	if alerts.Enabled() {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "???"
		}
		alerts.Warnf("Starting %s on %s port %s\nDiscovery: %v\nStatic: %d\nTrusted: %d\nReverseProxy: %v\nHost: %s\n",
			n.serverConfig.Name,
			hostname,
			getport(n.serverConfig.ListenAddr), !n.serverConfig.NoDiscovery, numStatic, numTrusted, n.config.RPCBehindProxy, n.config.HTTPVirtualHosts[0])
	}
	// Otherwise copy and specialize the P2P configuration
	services := make(map[reflect.Type]Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular service
		ctx := &ServiceContext{
			config:         n.config,
			services:       make(map[reflect.Type]Service),
			EventMux:       n.eventmux,
			AccountManager: n.accman,
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.services[kind] = s
		}
		// Construct and save the service
		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(service)
		if _, exists := services[kind]; exists {
			return &DuplicateServiceError{Kind: kind}
		}
		services[kind] = service
	}
	// Gather the protocols and start the freshly assembled P2P server
	for _, service := range services {
		running.Protocols = append(running.Protocols, service.Protocols()...)
	}
	if err := running.Start(ctx); err != nil {
		return convertFileLockError(err)
	}
	// Start each of the services
	started := []reflect.Type{}
	for kind, service := range services {
		// Start the next service, stopping all previous upon failure
		if service == nil {
			log.Warn("skipping service:", "kind", kind)
			continue
		}
		if err := service.Start(running); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}
			running.Stop()

			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}
	// Lastly start the configured RPC interfaces
	if err := n.startRPC(services, debug.AddLoop()); err != nil {
		for _, service := range services {
			service.Stop()
		}
		running.Stop()
		return err
	}
	// Finish initializing the startup
	n.services = services
	n.server = running
	n.stop = make(chan struct{})

	return nil
}

func (n *Node) openDataDir() error {
	if n.config.DataDir == "" {
		return nil // ephemeral
	}

	instdir := DefaultDatadirByChain(n.chaincfg)
	if err := os.MkdirAll(instdir, 0700); err != nil {
		return err
	}
	// Lock the instance directory to prevent concurrent use by another instance as well as
	// accidental use of the instance directory as a database.
	release, _, err := flock.New(filepath.Join(instdir, "LOCK"))
	if err != nil {
		return convertFileLockError(err)
	}
	n.instanceDirLock = release
	return nil
}

func parseAllowNet(allowIPmasks []string) netutil.Netlist {
	var allowIPMap netutil.Netlist
	for _, cidr := range allowIPmasks {
		if cidr == "*" {
			cidr = "0.0.0.0/0"
		}
		if cidr == "0.0.0.0/0" {
			log.Warn("Allowing public RPC access. Automatically enabling NO_KEYS=1 and NO_SIGN=1")

		}
		if !strings.Contains(cidr, "/") {
			cidr += "/32" // helper for single IPs
		}
		allowIPMap.Add(cidr)
	}
	return allowIPMap
}

// startRPC is a helper method to start all the various RPC endpoint during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) startRPC(services map[reflect.Type]Service, donefunc func()) error {
	defer donefunc()
	// gather allownet
	allownet := parseAllowNet(n.config.RPCAllowIP)

	// Gather all the possible APIs to surface
	apis := n.apis()
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}
	if len(apis) == 0 {
		return errors.New("no APIs offered by the services")
	}
	// Start the various API endpoints, terminating all in case of errors
	if err := n.startInProc(apis); err != nil {
		return err
	}
	if err := n.startIPC(apis); err != nil {
		n.stopInProc()
		return err
	}
	if err := n.startHTTP(n.httpEndpoint, apis, n.config.HTTPModules, n.config.HTTPCors, n.config.HTTPVirtualHosts, allownet, n.config.RPCBehindProxy); err != nil {
		n.stopIPC()
		n.stopInProc()
		return err
	}
	if err := n.startWS(n.wsEndpoint, apis, n.config.WSModules, n.config.WSOrigins, n.config.WSExposeAll, allownet, n.config.RPCBehindProxy); err != nil {
		n.stopHTTP()
		n.stopIPC()
		n.stopInProc()
		return err
	}
	// All API endpoints started successfully
	n.rpcAPIs = apis
	return nil
}

// startInProc initializes an in-process RPC endpoint.
func (n *Node) startInProc(apis []rpc.API) error {
	// Register all the APIs exposed by the services
	if n.config.NoInProc {
		return nil
	}
	debugRpc := sense.EnvBool("DEBUG_RPC")
	handler := rpc.NewServer()
	for _, api := range apis {
		if _, err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		if debugRpc {
			n.log.Debug("InProc registered", "service", fmt.Sprintf("%T ( %s_ )", api.Service, api.Namespace))
		}
	}
	n.inprocHandler = handler
	return nil
}

// stopInProc terminates the in-process RPC endpoint.
func (n *Node) stopInProc() {
	if n.inprocHandler != nil {
		n.inprocHandler.Stop()
		n.inprocHandler = nil
	}
}

// startIPC initializes and starts the IPC RPC endpoint.
func (n *Node) startIPC(apis []rpc.API) error {
	// Short circuit if the IPC endpoint isn't being exposed
	if n.ipcEndpoint == "" {
		return nil
	}
	debugRpc := sense.EnvBool("DEBUG_RPC")
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if _, err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return err
		}
		if debugRpc {
			n.log.Debug("IPC registered", "service", fmt.Sprintf("%T", api.Service), "namespace", api.Namespace)
		}
	}
	// All APIs registered, start the IPC listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = rpc.CreateIPCListener(n.ipcEndpoint); err != nil {
		return err
	}
	go func() {
		n.log.Info("IPC endpoint opened", "url", n.ipcEndpoint)

		for {
			conn, err := listener.Accept()
			if err != nil {
				// Terminate if the listener was closed
				n.lock.RLock()
				closed := n.ipcListener == nil
				n.lock.RUnlock()
				if closed {
					return
				}
				// Not closed, just some error; report and continue
				n.log.Error("IPC accept failed", "err", err)
				continue
			}
			go handler.ServeCodec("ipc", rpc.NewJSONCodec(conn), rpc.OptionMethodInvocation|rpc.OptionSubscriptions)
		}
	}()
	// All listeners booted successfully
	n.ipcListener = listener
	n.ipcHandler = handler

	return nil
}

// stopIPC terminates the IPC RPC endpoint.
func (n *Node) stopIPC() {
	if n.ipcListener != nil {
		n.ipcListener.Close()
		n.ipcListener = nil

		n.log.Info("IPC endpoint closed", "endpoint", n.ipcEndpoint)
	}
	if n.ipcHandler != nil {
		n.ipcHandler.Stop()
		n.ipcHandler = nil
	}
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (n *Node) startHTTP(endpoint string, apis []rpc.API, whitelistModules []string, cors []string, vhosts []string, allownet netutil.Netlist, behindreverseproxy bool) error {
	if len(allownet) == 0 && sense.Getenv("TESTING_TEST") != "1" {
		return fmt.Errorf("http rpc cant start with empty '-allowip' flag")
	}
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	log.Info("HTTP whitelist", "endpoint", endpoint, "modules", whitelistModules, "signing-enabled", !sense.IsNoSign(), "keystore-available", !sense.IsNoKeys())
	for _, module := range whitelistModules {
		whitelist[module] = true
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	//	var allMethods []string
	for _, api := range apis {
		if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			m, err := handler.RegisterName(api.Namespace, api.Service)
			if err != nil {
				return err
			}
			//			allMethods = append(allMethods, m...)
			n.log.Warn("HTTP method available", "methods", common.ToJson(m), "service", fmt.Sprintf("%T", api.Service), "namespace", api.Namespace)
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp4", endpoint); err != nil {
		return err
	}

	go rpc.NewHTTPServer(cors, vhosts, allownet, behindreverseproxy, handler).Serve(listener)
	n.log.Warn("HTTP endpoint opened", "usingReverseProxy", behindreverseproxy, "url", fmt.Sprintf("http://%s", endpoint), "cors", common.ToJson(cors), "vhosts", strings.Join(vhosts, ","), "allowip", allownet.String())

	// All listeners booted successfully
	n.httpEndpoint = endpoint
	n.httpListener = listener
	n.httpHandler = handler

	return nil
}

// stopHTTP terminates the HTTP RPC endpoint.
func (n *Node) stopHTTP() {
	if n.httpListener != nil {
		n.httpListener.Close()
		n.httpListener = nil

		n.log.Info("HTTP endpoint closed", "url", fmt.Sprintf("http://%s", n.httpEndpoint))
	}
	if n.httpHandler != nil {
		n.httpHandler.Stop()
		n.httpHandler = nil
	}
}

// startWS initializes and starts the websocket RPC endpoint.
func (n *Node) startWS(endpoint string, apis []rpc.API, modules []string, wsOrigins []string, exposeAll bool, allowedip netutil.Netlist, behindproxy bool) error {
	// Short circuit if the WS endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}
	if exposeAll {
		n.log.Warn("WebSocket exposing all modules over the network", "address", endpoint)
	}
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		if exposeAll || whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			m, err := handler.RegisterName(api.Namespace, api.Service)
			if err != nil {
				return err
			}
			n.log.Warn("WebSocket methods are available", "service", fmt.Sprintf("%T", api.Service), "namespace", api.Namespace, "methods", common.ToJson(m))
		}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp4", endpoint); err != nil {
		return err
	}
	go rpc.NewWSServer(wsOrigins, allowedip, behindproxy, handler).Serve(listener)
	n.log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%s", listener.Addr()))

	// All listeners booted successfully
	n.wsEndpoint = endpoint
	n.wsListener = listener
	n.wsHandler = handler

	return nil
}

// stopWS terminates the websocket RPC endpoint.
func (n *Node) stopWS() {
	if n.wsListener != nil {
		n.wsListener.Close()
		n.wsListener = nil

		n.log.Info("WebSocket endpoint closed", "url", fmt.Sprintf("ws://%s", n.wsEndpoint))
	}
	if n.wsHandler != nil {
		n.wsHandler.Stop()
		n.wsHandler = nil
	}
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.closemain != nil {
		n.closemain()
	}

	// Short circuit if the node's not running
	if n.server == nil {
		return ErrNodeStopped
	}

	// Terminate the API, services and the p2p server.
	n.stopWS()
	n.stopHTTP()
	n.rpcAPIs = nil
	failure := &StopError{
		Services: make(map[reflect.Type]error),
	}
	for kind, service := range n.services {
		if err := service.Stop(); err != nil {
			failure.Services[kind] = err
		}
	}
	n.server.Stop()
	n.services = nil
	n.server = nil

	// Release instance directory lock.
	if n.instanceDirLock != nil {
		if err := n.instanceDirLock.Release(); err != nil {
			n.log.Error("Can't release datadir lock", "err", err)
		}
		n.instanceDirLock = nil
	}

	// unblock n.Wait
	close(n.stop)

	// Remove the keystore if it was created ephemerally.
	var keystoreErr error
	if n.ephemeralKeystore != "" {
		keystoreErr = os.RemoveAll(n.ephemeralKeystore)
	}

	if len(failure.Services) > 0 {
		return failure
	}
	if keystoreErr != nil {
		return keystoreErr
	}
	n.stopIPC()
	return nil
}

// Wait blocks the thread until the node is stopped. If the node is not running
// at the time of invocation, the method immediately returns.
func (n *Node) Wait() {
	n.lock.RLock()
	if n.server == nil {
		n.lock.RUnlock()
		return
	}
	stop := n.stop
	n.lock.RUnlock()

	select {
	case <-stop:
		log.GracefulShutdown(fmt.Errorf("node is out"))
	case <-n.ctx.Done():
		n.Stop()
	}
}

// Restart terminates a running node and boots up a new one in its place. If the
// node isn't running, an error is returned.
func (n *Node) Restart() error {
	n.log.Info("Restarting P2P node")
	if err := n.Stop(); err != nil {
		return err
	}
	if err := n.Start(n.ctx); err != nil {
		return err
	}
	return nil
}

// Attach creates an RPC client attached to an in-process API handler.
func (n *Node) Attach(ctx context.Context, name string) (*rpcclient.Client, error) {
	n.log.Trace("Attaching new client", "name", name, "caller2", log.Caller(1), "caller3", log.Caller(2))
	if n.config.NoInProc {
		return nil, fmt.Errorf("inproc disabled")
	}
	n.lock.RLock()
	defer n.lock.RUnlock()

	if n.server == nil {
		return nil, ErrNodeStopped
	}
	cl := rpcclient.DialInProc(ctx, n.inprocHandler)
	cl.Name = name
	return cl, nil
}

// RPCHandler returns the in-process RPC request handler.
func (n *Node) RPCHandler() (*rpc.Server, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if n.inprocHandler == nil {
		return nil, ErrNodeStopped
	}
	return n.inprocHandler, nil
}

// Server retrieves the currently running P2P network layer. This method is meant
// only to inspect fields of the currently running server, life cycle management
// should be left to this Node entity.
func (n *Node) Server() *p2p.Server {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.server
}

// Service retrieves a currently running service registered of a specific type.
func (n *Node) Service(service interface{}) error {
	n.lock.RLock()
	defer n.lock.RUnlock()

	// Short circuit if the node's not running
	if n.server == nil {
		return ErrNodeStopped
	}
	// Otherwise try to find the service to return
	element := reflect.ValueOf(service).Elem()
	if running, ok := n.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrServiceUnknown
}

// DataDir retrieves the current datadir used by the protocol stack.
// Deprecated: No files should be stored in this directory, use InstanceDir instead.
func (n *Node) DataDir() string {
	return n.config.DataDir
}

func (n *Node) Config() Config {
	return *n.config
}

// InstanceDir retrieves the instance directory used by the protocol stack.
func (n *Node) InstanceDir() string {
	return n.config.instanceDir()
}

// AccountManager retrieves the account manager used by the protocol stack.
func (n *Node) AccountManager() *accounts.Manager {
	return n.accman
}

// IPCEndpoint retrieves the current IPC endpoint used by the protocol stack.
func (n *Node) IPCEndpoint() string {
	return n.ipcEndpoint
}

// HTTPEndpoint retrieves the current HTTP endpoint used by the protocol stack.
func (n *Node) HTTPEndpoint() string {
	return n.httpEndpoint
}

// WSEndpoint retrieves the current WS endpoint used by the protocol stack.
func (n *Node) WSEndpoint() string {
	return n.wsEndpoint
}

// EventMux retrieves the event multiplexer used by all the network services in
// the current protocol stack.
func (n *Node) EventMux() *event.TypeMux {
	return n.eventmux
}

// OpenDatabase opens an existing database with the given name (or creates one if no
// previous can be found) from within the node's instance directory. If the node is
// ephemeral, a memory database is returned.
func (n *Node) OpenDatabase(name string, cache, handles int) (aquadb.Database, error) {
	if n.config.DataDir == "" {
		return aquadb.NewMemDatabase(), nil
	}
	return aquadb.NewLDBDatabase(n.config.resolvePath(name), cache, handles)
}

// ResolvePath returns the absolute path of a resource in the instance directory.
func (n *Node) ResolvePath(x string) string {
	return n.config.resolvePath(x)
}

// apis returns the collection of RPC descriptors this node offers.
func (n *Node) apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(n),
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPublicAdminAPI(n),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   debug.Handler,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(n),
			Public:    true,
		}, {
			Namespace: "web3",
			Version:   "1.0",
			Service:   NewPublicWeb3API(n),
			Public:    true,
		},
	}
}
