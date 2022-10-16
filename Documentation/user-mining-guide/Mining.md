# Mining

## Hashrate Benchmarks

See `#hashrate-reports` channel for many more: [discord](https://discord.com/invite/J7jBhZf)

## New Mining Software

There are many varieties of aquachain mining software.

As of this edit, latest is [aquaminer-gpu](https://github.com/aquachain/aquaminer-gpu)

Compiling is easy.

```
git clone --branch linux --recurse-submodules https://github.com/aquachain/aquaminer-gpu
cd aquaminer-gpu
make
ls bin/
```

Running all the miners are generally the same. The `-F` flag and then the pool URL, or your node URL if solo mining.

## Aquachain Miner Resources

- [Network Pool Status](https://aquacha.in/status/miners)

## Pool mining

Choose a pool from those listed on [pool status](https://aquacha.in/status/miners) or [aquachain.github.io](https://aquachain.github.io/explorer/#/pool)

Run the miner with the correct flags, considering making a short script

`./aquaminer.exe -F http://<pooladdress>:8888/<address>/<worker>`

Here replace `<address>` with your wallet address and `<worker>` with any custom name for the CPU you are using with your address. Remember multiple workers can be used with a single wallet address and in this case all paid money will go to the same wallet. And swap `<pooladdress>` for the actual pool host.

For CPU, use `-t` flag for number of cpu (default all)

`./aquaminer.exe -F http://pool.aquachain-foundation.org:8888/<address>/<worker>`

For GPU, use `-d` flag to choose the OpenCL device number
  ```
  pooladdr='http://pool.whatever:8888/<address>/<worker>'
  ./aquacppminer-gpu -d 0 -F ${pooladdr} &
  ./aquacppminer-gpu -d 1 -F ${pooladdr} &
  ```
Since these are now running in background, use something like "killall aquacppminer-gpu" to kill all the miners.

### Pools

These are the currently known pools, edit to add more: [pools.json](https://github.com/aquachain/aquachain.github.io/blob/master/pools.json)

[aquachain explorer: pools](https://aquachain.github.io/explorer/#/pool)

[multipool status](https://info.aquacha.in/status/miners)

## Solo mining:

- **Don't solo mine unless you can actually mine blocks once in a while, start at a pool**

- **Dont keep your keys on your RPC server**

- **Use the `-aquabase` flag to set coinbase with no private key on server**

Be sure to **wait and sync before mining**. It doesn't take long.

To reduce orphan blocks, also be sure to **have peers** and check a block explorer to see the current block number and hash.

Do not key any keys inside the "nokeys" directory. You can safely delete `aquaminingdata` and `nokeys` (make sure you dont keep keys in there!)

**Run your RPC server like so: `aquachain -rpc -rpcaddr 192.168.1.5 -datadir aquaminingdata -keystore nokeys -aquabase 0x3317e8405e75551ec7eeeb3508650e7b349665ff`**

Later, to spend and use the AQUA console, just double click aquachain. This way, you keep your keys safe (in the default keystore dir) and don't mix `datadir`, this can prevent RPC attacks.

Please see the many cases where people have lost their ETH because leaving RPC open for even one minute.

### Solo farm

This assumes your AQUA node will be running from LAN 192.168.1.3, with other workers on the same lan.

WORKERS: `aquacppminer --solo -F http://192.168.1.3:8543/`

Also consider running a pool! [open-aquachain-pool](https://github.com/aquachain/open-aquachain-pool/blob/master/docs/TUTORIAL.md)

and see mining proxy: [aquachain-proxy](https://github.com/rplant8/aquachain-proxy)

**Coinbase**

This is the address that receives the block reward. The mining node does not need the key.

Use the `-aquabase` flag, or from console:

```
miner.setAquabase('0x.your address..')
```

