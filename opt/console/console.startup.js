// console.startup.js
// this file is embedded and executed on boot
// it might look like web.version.node exists but its a getter
// lets call getinfo once and format the output

// friendly name for algo
function algoname(version) {
    switch (version) {
        case 1: return "Ethash";
        case 2: return "Argon2id";
        case 3: return "Argon2id-B";
        case 4: return "Argon2id-C";
        default: return "Unknown";
    }
}

// fetch the info from the node
function getinfo() {
    try {
        var instance = web3.version.node;
    } catch (e) {
        console.log("error getting instance: " + e);
        var instance = "unknown";
    }
    var info = {
        "instance": instance,
        "chainId": Number(aqua.chainId(), 16), // hex->dec
        "gasPrice": web3.fromWei(aqua.gasPrice, 'gwei'),
    }
    try {
        info["coinbase"] = aqua.coinbase;
    } catch (e) {
        console.log("getting coinbase:", e);
        info["coinbase"] = undefined;
     }
     var head = aqua.getBlock('latest');
     var headinfo = {
         "block": head.number,
         "timestamp": head.timestamp,
         "hash": head.hash,
         "gasLimit": head.gasLimit,
         "difficulty": head.difficulty,
         "algo": head.version,
         "algoname": algoname(head.version)
        };
    info["headinfo"] = headinfo;

    try {
        this.admin && (info["datadir"] = this.admin.datadir);
    } catch (e) { }

    return info;
}


function welcome() {
    var info = getinfo();
    var headinfo = info.headinfo;

    console.log("instance:   " + info.instance);
    console.log("at block:   " + headinfo.block + " (" + new Date(1000 * headinfo.timestamp) + ")");
    console.log("    head:   " + headinfo.hash);
    console.log("coinbase:   " + info.coinbase);
    console.log("  gasPrice: " + info.gasPrice + " gigawei");
    console.log("  gasLimit: " + headinfo.gasLimit + " units");
    var diffstr = "";
    if (headinfo.difficulty < 1000) {
    console.log("nextsigner: " + headinfo.difficulty);
    } else {
    console.log("difficulty: " + (headinfo.difficulty / 1000000.0).toFixed(2) + " MH");
    }
    console.log("   chainId: " + info.chainId);
    console.log("      algo: " + headinfo.algo + " (" + headinfo.algoname + ")");
    if (info.datadir !== undefined) {
    console.log("   datadir: " + info.datadir);
    }
}

if (true) {
    try {
    welcome();
    } catch (e) {
        console.log("error in welcome: " + e);
    }
}