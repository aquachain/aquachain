# How to Host a Public RPC

To set up a private RPC, run with `-rpc` and/or `-ws` flags. To make it public, either forward it using a reverse proxy, or use `-rpcaddr` and `wsaddr` flags to bind to a public interface.

For serving public RPC, you will want to disable keystore and signing, and keep private keys away from it in general. We have 'NO_SIGN' and 'NO_KEYS' environmental variables for this purpose.

```bash
# for direct connections to the internet without reverse proxy
NO_SIGN=1 NO_KEYS=1 aquachain -rpc -rpcaddr 0.0.0.0 -rpcvhosts '*' -ws -wsorigins '*' -wsaddr 0.0.0.0 -aquabase '0x1234...5678' -allowip 0.0.0.0/0 
```

It is better to run behind a proxy, such as caddy, thatcan serve https and handle rate limiting.

```bash
# for running behind reverse proxy (listens on 127.0.0.1, port 8543/8544)
NO_SIGN=1 NO_KEYS=1 aquachain -rpc -rpcvhosts '*' -ws -wsorigins '*' -aquabase '0x1234...5678' -allowip 0.0.0.0/0 -behindproxy
```

```bash
# first, clone the explorer website
mkdir -p /var/www/aqua-explorer
git clone https://github.com/aquachain/explorer.git /var/www/aquachain/explorer
# edit the endpoints config
editor /var/www/aquachain/explorer/endpoints.json
```

Here is a Caddyfile snippet for a public reverse proxy with explorer GET

```caddy
aquachain.example.com {
    reverse_proxy /rpc localhost:8543 {
        header_up Content-Type application/json
    }
    reverse_proxy /ws localhost:8544
    file_server /explorer/* {
        root /var/www/aquachain/explorer
    }
}        
```





