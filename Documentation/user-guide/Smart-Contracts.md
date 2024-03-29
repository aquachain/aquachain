# Smart Contracts

Something that sets Aquachain apart from other cryptocurrency is that we are able to write programs that exist on our blockchain.

AQUA uses the same stack based VM as Ethereum, and is compatible with all ETH smart contracts.

The use of tokens are discouraged. Think bigger. 
There are some legitimate use-cases for the ERC20 standard, but for Aquachain, the ERC721 standard makes more sense to deploy. 

Important: When deploying, use "Byzantium" EVM Target and Optimize code so it fits in the block.

Some opcodes do not exist here, such as CHAINID and CREATE2, and if your contract depends on these you must modify them.


## Simple Example


Below is an example contract deployed on the aquachain at [0xF179C8ec4cE31d8B9f16fA12c88A6091fD06d62a](0xF179C8ec4cE31d8B9f16fA12c88A6091fD06d62a)

Copy paste this in your browser for [live demo](https://aquachain.github.io/explorer/#/contract/0xF179C8ec4cE31d8B9f16fA12c88A6091fD06d62a/double(int256)/100) (see Return Data: uint256): `https://aquachain.github.io/explorer/#/contract/0xF179C8ec4cE31d8B9f16fA12c88A6091fD06d62a/double(int256)/100`


It is a simple contract that demonstrates two things:

1. Writing a contract is easy. Just send a transaction containing Data, with no recipient. Add some fuel, and wooo! It's deployed.

```
aqua.sendTransaction({from:aqua.coinbase,data:'6060604052341561000f57600080fd5b60b18061001d6000396000f300606060405260043610603f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa146044575b600080fd5b3415604e57600080fd5b606260048080359060200190919050506078565b6040518082815260200191505060405180910390f35b60008160020290509190505600a165627a7a723058208aa56e39b6d6a9caab4b9a9dc5241ea1c56dd40cf77f1c1e66af80c59fef24640029'})
```

2. Writing a secure contract is not easy. Here is a **dangerous overflow**: (Input: `9999999999999999999999999999999999999999999999999999999999999999999999999999` returns `-95792089237316195423570985008687907853269984665640564039457584007913129639938`

### Source Code

**The following is buggy code**

The above contract was made with solidity language, compiled on the [Remix](https://remix.ethereum.org) browser compiler.

```
pragma solidity ^0.4.18;

contract Double {
    function double(int a) public pure returns(int) { return 2*a;} // buggy
}
```

using solc or Remix browser compiler, compiles to bytecode , which is sent in a transaction in the *data* field (no "to" field, or "zero address"):

```
6060604052341561000f57600080fd5b60b18061001d6000396000f300606060405260043610603f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa146044575b600080fd5b3415604e57600080fd5b606260048080359060200190919050506078565b6040518082815260200191505060405180910390f35b60008160020290509190505600a165627a7a723058208aa56e39b6d6a9caab4b9a9dc5241ea1c56dd40cf77f1c1e66af80c59fef24640029
```

and an ABI, to let us know what functions are available. in this case, `double(int256)` that returns an `int256`

```
[
	{
		"constant": true,
		"inputs": [
			{
				"name": "a",
				"type": "int256"
			}
		],
		"name": "double",
		"outputs": [
			{
				"name": "",
				"type": "int256"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	}
]

```
