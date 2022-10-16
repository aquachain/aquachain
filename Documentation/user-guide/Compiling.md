# Building from the source

You will need the latest version of the Go programming language.

Here we are using go1.19.2, there is likely a [newer version of go](https://golang.org/dl/), use that instead.

### Installing Go on Linux

```
wget https://go.dev/dl/go1.19.2.linux-amd64.tar.gz
sudo tar xvaf go1.19.2.linux-amd64.tar.gz -C /usr/local/
sudo ln /usr/local/go/bin/go* /usr/local/bin/
```

### Installing Go on Apple macOS

```
wget https://go.dev/dl/go1.19.2.darwin-amd64.tar.gz
sudo tar xvaf go1.19.2.darwin-amd64.tar.gz -C /usr/local/
sudo ln /usr/local/go/bin/go* /usr/local/bin/
```

### Checking Go version

```
which go
go version
```

## Downloading The Source

```
git clone https://gitlab.com/aquachain/aquachain
cd aquachain
```

## Compiling

In the base directory of the repository, you can run a variety of 'make' targets.

When finished compiling, they are in the `./bin` directory.

build the Aquachain command (p2p node, rpc server)

```
make
```

it should pop out as ./bin/aquachain or ./bin/aquachain.exe if on windows.

### Customized Build

or build any of the available targets

```
all           crossold      goget         race
bin/          default       hash          release
bootnode      devtools      help          release/
checkrelease  docs          install       test
clean         echoflags     linter
cross         generate      package

```

## Cross Compilation

This is how releases can be made, using pure Go toolchain and no C dependencies.

```
make clean release release=1

```

## Contributing

If you are contributing, you will want to 'fork' the main repo on github, and add your fork like so, changing `your-name` and `patch-1` to whatever you need:

```
git remote add fork git@github.com:yourname/aquachain.git
git checkout -b patch-1
```

When done making commits, use `git push fork patch-1` and either open a pull request or ask to merge.

During development on your branch there may be many commits on the `master` branch. You can re-synchronize by using `git pull -r origin master` or `git rebase -i` to avoid needing _Merge commits_.

