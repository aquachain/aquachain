# Storage

Storing things on the blockchain is forever. It is best to avoid storing data on the blockchain unless you need to for example censorship resistance.

Instead, store a hash as reference to a file hosted elsewhere.

## IPFS

"InterPlanetary File System is a protocol and network designed to create a content-addressable, peer-to-peer method of storing and sharing hypermedia in a distributed file system."

With IPFS you are able to upload a file, and retrieve it (from somewhere else) using the file's hash.

Use the short IPFS "CID" on-chain, instead of storing the data directly on-chain.

Then, host an IPFS node and your file will be online forever. Services exist that offer IPFS hosting, and some are free.

