# How to Host a Public RPC

To set up a private RPC, run with `-rpc` flag.

To set up a public RPC, follow this guide.

### About Public RPC Nodes

The network is able to function without any public RPC nodes, but they add convenience to end-users.

What they do is provide a JSON RPC over HTTP(s). Applications such as explorers and wallets can use public RPCs to fetch data and submit transactions.

Public RPC Nodes do not need any private keys. They should not be on the same machine as private keys.

Currently, the aquachain command doesn't use TLS/HTTPS to provide a secure RPC. For now, it is necessary to use a reverse proxy for this purpose.

### The setup

Here is *one of many* setups that can provide a public https endpoint, offering a public RPC for the world to use.

  * For SSL (recommended), setup your subdomain DNS to your IP, before this.
  * It is recommended to use a machine with 2GB or more RAM.
  * Need at least 50GB disk space, recommended SSD but not necessary.
  * Preferably a dedicated machine, such as a VPS with no other uses.
  * A newer version of `caddy` or `go` may have arrived since this was published.

You can follow this guide which uses a fresh VPS. The OS is Ubuntu.

All commands as root user... lets go!

```
# add users
adduser --system aqua
adduser --system caddy

# install go (can skip if download aquachain binary)
mkdir -p /root/dl
cd /root/dl
wget -4 'https://golang.org/dl/go1.15.6.linux-amd64.tar.gz'
tar xvf go1.15.6*.tar.gz -C /usr/local/
ln -s /usr/local/go/bin/* /usr/local/bin/

# install caddy (reverse proxy, ssl, web server)
cd /root/dl
wget -4 -O /usr/local/bin/caddy 'https://caddyserver.com/api/download?os=linux&arch=amd64'
chmod +x /usr/local/bin/caddy
setcap CAP_NET_BIND_SERVICE=+eip /usr/local/bin/caddy

# setup clean reboots for database health
wget -4 -O /etc/rc0.d/K01aquachain https://github.com/aquachain/aquachain/raw/master/contrib/K01aquachain
chmod +x /etc/rc0.d/K01aquachain

# setup aquachain rpc
cd /home/aqua
sudo -u aqua git clone https://gitlab.com/aquachain/aquachain src/aquachain
cd src/aquachain
sudo -u aqua make
mv /home/aqua/src/aquachain/bin/aquachain /usr/local/bin/aquachain

# setup aqua reboot
cat <<EOF >/home/aqua/reboot.bash
#!/bin/bash
TERM=xterm
# can modify these for example --config or something
AQUAFLAGS="-nokeys -gcmode archive -rpc -rpccorsdomain='*' -rpcvhosts='*'"
tmux new-session -n aqua -d /usr/local/bin/aquachain $AQUAFLAGS
EOF

chmod +x /home/aqua/reboot.bash
echo '@reboot bash /home/aqua/reboot.bash' | crontab -u aqua

# setup caddy reverse proxy
cd /home/caddy
wget -4 https://github.com/aquachain/aquachain/raw/master/contrib/Caddyfile
echo "/usr/local/bin/caddy start" >> /home/caddy/reboot.bash
chmod +x /home/caddy/reboot.bash
echo '@reboot bash /home/caddy/reboot.bash' | crontab -u caddy
```

### Now customize the Caddyfile with your domain name

Don't forget to edit /home/caddy/Caddyfile and replace the dummy domain name.

### Putting it all together

Now you have a machine that will automatically launch caddy and aquachain, accepting secure requests from anyone on the internet. The machine has no keys, never uses keys, never signs anything.

If this is all you are using the server for, you are probably done with your setup.

Restart the VPS machine. (as root, `reboot`)

Open up a terminal and run: `aquachain attach https://mydomain.examplename`

Use your domain name instead of the dummy name.

If you get an AQUA console, you have achieved your goal, a public https rpc server..

