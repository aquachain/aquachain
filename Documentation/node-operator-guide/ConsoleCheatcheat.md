# Console Cheatsheet

### Node Operators

read aquachain help to see all flags

use `aquachain [your flags] dumpconfig > /tmp/aqua.config` to create a new config that shows additional options.

make sure you somehow send SIGINT to your aquachain node before system shutdown.
otherwise, your node will have to resync a number of blocks.

with your node running, even without -rpc flag, 
you can run `aquachain attach` and connect to your instance through ipc socket.

here are some useful snippets

```
// use now() anywhere you need a timestamp
now = function () { return Math.floor(new Date().getTime() / 1000)}

// for example exporting chain to a timestamped file
admin.exportChain('aquachain-bootstrap'+now()+'.dat')
```


### Mining Node Operators

sometimes you'd like to change something without restarting the server

```
// change minimum gas price for inclusion (local txs can still be 0)
// use -gasprice flag, example 1000000000
miner.setGasPrice(web3.toWei(1, 'gwei'))

// change the extradata included in block
// use -extra flag
miner.setExtra("hello world")
```

