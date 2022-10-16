# How to run an archive node

Estimated time: 10-60 minutes depending on hardware and network connection.

If *starting from a fresh unsynced node*, simply run the aquachain node with `-gcmode archive` flag.

If *your node is already synchronized*, you will need to re-sync the entire chain in archive mode.

To complete this step offline, you can export the chain to a file and clear the node's mainnet database,
then import that file.

1. `aquachain export aqua-mainnet.dat` and then `aquachain removedb`
2. Run `aquachain -gcmode archive import mainnet.dat`
3. Now, run aquachain with `-gcmode archive` flag.
