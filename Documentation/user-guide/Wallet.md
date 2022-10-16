# Wallet

You need to have a wallet to acquire AQUA. A Wallet provides access to your crypto which live on the AQUA blockchain aka Aquachain. This crypto is only available through your private key. If you lose this secret/private key, there is no way to recover your assets. Please safely store the wallet keys in a secure manner. Anyone with access to your private keys gains access to your crypto and can perform outward transactions. This gave birth to the famous expression “not your (private) keys, not your crypto”.

Note: To safeguard your private keys, its best to use a hardware wallet instead of storing it on your device or paper. Using a hardware wallet is the most secure way of accessing your tokens because your signing key never leave the device.

When you're setting up a wallet, be sure to:
✅ Download and install only the latest version from an official source.
✅ Follow the setup guide carefully.
✅ Safely back up your recovery phrases.
❌ NEVER share your recovery phrases with anyone, under any circumstances.
❌ NEVER input your recovery phrase to a website or app, other than your wallet app.

## Metamask

Metamask is a cryptocurrency wallet available for IOS, Android and as a Chrome extension. Once installed, create a new wallet. Make sure to securely backup your private key. Anyone with access to your private keys gains access to your crypto.
By default, MetaMask creates a wallet on Ethereum Mainnet network. To use AQUA with MetaMask, you need to add AquaChain network. Go to Settings->Networks->Add a network and fill following fields

```
Network Name: Aquachain Mainnet
RPC URL: https://c.onical.org (or yours)
Chain ID: 61717561
Symbol: AQUAC (use this to get correct cryptocompare api response)
Block Explorer URL: https://aquachain.github.io/explorer/#/
```

After saving above network information, switch to Aquachain network and you are ready to use MetaMask with AQUA.

MetaMask also supports hardware wallets such as Ledger and Trezor. See this [guide](https://metamask.zendesk.com/hc/en-us/articles/360020394612-How-to-connect-a-Trezor-or-Ledger-Hardware-Wallet) on how to set up MetaMask to use your hardware wallet.

## Frame Wallet

"A privacy focused Ethereum wallet that runs natively on macOS, Windows and Linux"

Get here [frame.sh](https://frame.sh) and [frame releases](https://github.com/floating/frame/releases)

Configure same as metamask above

## Full node wallet

### Advanced mode

Such usecases for a full node wallet include:
  * if you wish to programmatically sending transactions using `personal_sendTransaction` 
  * if you only want to use wallet through the secure console wallet

Both situations require a secure network, and are for advanced users only. Consider using a common wallet.

<em>If you use a JSON Keyfile wallet attached to your node, you MUST take safety precautions.</em>

It is BETTER to use `eth_sendRawTransaction` to submit a signed transaction, and keep your key away from your RPC node.

Anyone can set up complex private aquachain networks with multiple nodes, some with keys, some without, and transactions will be broadcast to the main network if at least one of the nodes are connected to the "world". Consider keeping your signing node "offline" if possible, and use `personal_signTransaction` which returns a signed tx without broadcasting it.

**You do NOT need a private key to run an aquachain node**

But, a password-protected JSON keyfile is perfectly safe if kept secured, and securely backed up.
The most common issue with this type of wallet is that it can't be easily written down in its current form.
Most users will probably want to NOT use this, and instead use a mnemonic phrase.

### Generating new console wallet

A wallet can be generated in three ways from the aquachain command:

- `aquachain.exe account new`
- `aquachain.exe paper 10`
- in the AQUA Console: `personal.newAccount()`

### Send a transaction

Start a transaction by typing `send` and press enter (type `n` or `ctrl+c` to cancel)

### Most important commands

Check balance with `aqua.balance(aqua.coinbase)`

Check all accounts balances: `balance()`

Send a transaction (easy way): `send`

Also see the [Console Cheatsheet](../../node-operator-guide/ConsoleCheatcheat/) and other node-operator-guide resources, since you operate a node.

### Example: Sending a transaction the hard way

Before sending coins, you must unlock the account:

`personal.unlockAccount(aqua.accounts[0])`

This next command will send 2 AQUA from "account 0" (first account created) to the typed account below:

```
aqua.sendTransaction({from: aqua.accounts[0], to: '0x3317e8405e75551ec7eeeb3508650e7b349665ff', value:web3.toWei(2,"aqua")});
```

Since its a javascript console, you can do something like this:

```
var destination = '0x3317e8405e75551ec7eeeb3508650e7b349665ff';
aqua.sendTransaction({from: aqua.accounts[0], to: destination, value:web3.toWei(2,"aqua")});
```

Default gas price is 0.1 gwei, if you want to specify a higher gas price you can add it to the tx:

```
var tx = {from: aqua.accounts[0], to: destination, gasPrice: web3.toWei(20, "gwei"), value:web3.toWei(2,"aqua")};
aqua.sendTransaction(tx);
```

### Useful Console Commands

(Save time! Press tab twice to auto-complete everything.)

```
balance()
aqua.balance(aqua.coinbase)
send
admin.nodeInfo.enode
net.listening
net.peerCount
admin.peers
aqua.coinbase
aqua.getBalance(aqua.coinbase)
personal
aqua.accounts
miner.setAquabase(web3.aqua.accounts[0])
miner.setAquabase(“0x0000000000000000000000000000000000000000”)
miner.start()
miner.stop()
miner.hashrate
aqua.getBlock(0)
aqua.getBlock(“latest”)
aqua.blockNumber
web3.aqua.getBlock(BLOCK_NUMBER).hash
aqua.syncing
debug.verbosity(6) // highest logging level, 3 is default
```

### Import Private Key

If you created a wallet with `aquachain.exe paper` or `aquachain wallet` you will have an unprotected private key. To send coins from the account derived from this key, you must import it into the console.

to import a private key into the aquachain console:

prepare your private key as a **simple text file with no spaces**. name it anything.txt for example

run `aquachain.exe account import anything.txt`

<b>make sure to delete/shred the private key txt file!!!</b>


### Backup Private Key

Running `aquachain account list` will list all key locations. Simply copy these files and check that directory.
