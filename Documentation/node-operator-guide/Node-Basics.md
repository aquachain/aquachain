# Full Node


#### **First Boot**

To get started, build (see [Compiling](Compiling)) or download the latest release **only from** https://github.com/aquachain/aquachain/releases

Unzip the archive, and double click the `aquachain.exe` application to open the javascript console.

When you open the program, you immediately start finding nodes and synchronizing with the network. It should take under 1 hour to sync the entire chain at this time.

## Startup Options

Some times you might need to start aquachain with a certain command, or flags.

Here we say `aquachain.exe` but if you are on linux, use `aquachain`.


### Example command arguments

```
aquachain.exe -rpc -rpccorsdomain '*' # serve 127.0.0.1:8543 rpc server
aquachain.exe -rpc daemon # rpc, no console
aquachain.exe account new # create new json keyfile
aquachain.exe -h # show help
aquachain.exe version # show version
aquachain.exe removedb # delete blockchain (keeping private keys)
aquachain.exe -your-flags dumpconfig > my.aqua.config.toml
aquachain.exe -config my.aqua.config.toml
aquachain.exe paper 10 ## generates ten addresses
aquachain.exe paper -vanity 123 10 ## generates ten addresses beginning with 0x123
```

### Disabling P2P

To disable p2p and discovery, use the `-offline` flag.
This is useful if you just want to use the AQUA Console to analyze your current blockchain status, or sign raw transactions offline.

## Aquachain Console

You know if you are in the aquachain console if you see the **AQUA>** prompt.
It is a (basic) javascript console, with (tab) auto-complete and (up/down) command history.

- Load a local script with the `loadScript('filename')` function.
- List accounts with `aqua.accounts`
- Check all balances with `balance()`

