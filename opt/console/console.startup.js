//{
function algoname(version) {
    switch (version) {
        case 1: return "Ethash";
        case 2: return "Argon2id";
        case 3: return "Argon2id-B";
        case 4: return "Argon2id-C";
        default: return "Unknown";
    }
}

function getinfo() {
    var head = aqua.getBlock('latest');
    var info = {
        "instance": web3.version.node,
        "block": head.number,
        "timestamp": head.timestamp,
        "hash": head.hash,
        "coinbase": aqua.coinbase,
        "gasPrice": web3.fromWei(aqua.gasPrice, 'gwei'),
        "gasLimit": head.gasLimit,
        "difficulty": head.difficulty,
        "chainId": Number(aqua.chainId(), 16), // hex->dec
        "algo": head.version,
        "algoname": algoname(head.version)
    };
    return info;
}


function welcome() {
    var info = getinfo();
    console.log("instance: " + info.instance);
    console.log("at block: " + info.block + " (" + new Date(1000 * info.timestamp) + ")");
    console.log("  head: " + info.hash);
    console.log("coinbase:  " + info.coinbase);
    console.log("  gasPrice: " + info.gasPrice + " gigawei");
    console.log("  gasLimit: " + info.gasLimit + " units");
    console.log("  difficulty: " + (info.difficulty / 1000000.0).toFixed(2) + " MH");
    console.log("  chainId: " + info.chainId);
    console.log("    algo: " + info.algo + " (" + info.algoname + ")");
    try {
        this.admin && console.log(" datadir: " + admin.datadir);
    } catch (e) { }
    try {
        this.admin && console.log("  client: " + admin.clientVersion);
    } catch (e) { }
}

function oldwelcome() {
    var head = aqua.getBlock('latest');
    version = head.version
    console.log("instance: " + web3.version.node);
    console.log("at block: " + head.number + " (" + new Date(1000 * head.timestamp) + ")");
    console.log("  head: " + head.hash);
    try {
        var coinbase = aqua.coinbase;
        console.log("coinbase:  " + coinbase);
    } catch (e) { }
    console.log("  gasPrice: " + web3.fromWei(aqua.gasPrice, 'gwei') + " gigawei");
    console.log("  gasLimit: " + head.gasLimit + " units");
    console.log("  difficulty: " + (head.difficulty / 1000000.0).toFixed(2) + " MH");
    console.log("  chainId: " + Number(aqua.chainId(), 16).toString());
    console.log("    algo: " + version + " (" + algoname(version) + ")");

    try {
        this.admin && console.log(" datadir: " + admin.datadir);
        this.admin && console.log("  client: " + admin.clientVersion);
    } catch (e) { }
}

if (false) {
    welcome();
}