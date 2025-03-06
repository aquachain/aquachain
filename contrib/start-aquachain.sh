#!/bin/bash
set -ex
if [ "$1" = "stop" ]; then
    echo cant stop
    exit 1
fi
if [ -f /etc/default/aquachain ]; then
    . /etc/default/aquachain
fi
if [ -f /etc/aquachain/aquachain.conf ]; then
    . /etc/aquachain/aquachain.conf
fi
export JSONLOG
export NO_SIGN
export NO_KEYS
export COLOR
if [ -z "${RPC_ALLOW_IP}" ]; then
    RPC_ALLOW_IP="${RPCALLOWIP}"
fi
export RPC_ALLOW_IP
# Aquachain coinbase address
AQUABASE=${AQUABASE}

# Aquachain data directory (default: ~/.aquachain/
AQUACHAIN_DATADIR=${AQUACHAIN_DATADIR}

# Aquachain chain (mainnet, testnet, testnet3)
AQUACHAIN_CHAIN=${AQUACHAIN_CHAIN}

# Aquachain verbosity
VERBOSITY=${VERBOSITY-3}

# Additional arguments for Aquachain (added below)
AQUACHAIN_ARGS=${AQUACHAIN_ARGS}

# add -rpc and -ws flags
USE_RPC=1
# add -rpchost and -wshost flags
PUBLIC_RPC_MODE=0

if [ "${RPC_ALLOW_IP}" = "none" ]; then
    RPC_ALLOW_IP=""
    USE_RPC=0
fi
if [ "${RPC_ALLOW_IP}" = "0.0.0.0/0" ]; then
    USE_RPC=0
    PUBLIC_RPC_MODE=1
    export NO_KEYS=1
    export NO_SIGN=1
    echo warn: public rpc mode enabled, no keys or signing allowed
    echo warn: public rpc mode enabled, no keys or signing allowed 1>&2
fi
if [ -n "${RPC_ALLOW_IP}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --allowip \"${RPC_ALLOW_IP}\""
fi
if [ -n "${AQUACHAIN_DATADIR}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --datadir \"${AQUACHAIN_DATADIR}\""
fi
if [ -n "${AQUABASE}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --aquabase \"${AQUABASE}\""
fi
if [ -n "${AQUACHAIN_CHAIN}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} -chain ${AQUACHAIN_CHAIN}"
fi
if [ -n "${VERBOSITY}" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} -verbosity ${VERBOSITY}"
fi
# use public rpc
if [ "${PUBLIC_RPC_MODE}" = "1" ]; then
    echo "Serving public HTTP and WS RPC on all interfaces" 1>&2
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --rpc --rpcaddr 0.0.0.0 --ws --wsaddr 0.0.0.0"
fi
# use default localhost rpc
if [ "${USE_RPC}" = "1" ]; then
    AQUACHAIN_ARGS="${AQUACHAIN_ARGS} --rpc --ws"
fi
export AQUACHAIN_ARGS=${AQUACHAIN_ARGS}
echo "Starting Aquachain node with args: ${AQUACHAIN_ARGS}" 1>&2

# lol TODO: fix this arg expansion
cmdline=$(echo exec /usr/local/bin/aquachain ${AQUACHAIN_ARGS} daemon)
exec /bin/sh -c "${cmdline}"