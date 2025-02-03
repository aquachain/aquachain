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

function welcome() {
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


welcome();