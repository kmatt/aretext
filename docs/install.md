Install
=======

Supported Platforms
-------------------

-	Linux
-	macOS
-	FreeBSD
-	Windows 10 and later (build from source)

Official Binaries
-----------------

You can download the official binaries for Linux, macOS, and FreeBSD from [the aretext releases page](https://github.com/aretext/aretext/releases).

### Linux x86 64-bit

```
VERSION=1.7.0
RELEASE=aretext_v${VERSION}_linux_amd64
curl -LO https://github.com/aretext/aretext/releases/download/v$VERSION/$RELEASE.tar.gz
tar -zxvf $RELEASE.tar.gz
sudo cp $RELEASE/aretext /usr/local/bin/
```

### Linux ARM 64-bit

```
VERSION=1.7.0
RELEASE=aretext_v${VERSION}_linux_arm64
curl -LO https://github.com/aretext/aretext/releases/download/v$VERSION/$RELEASE.tar.gz
tar -zxvf $RELEASE.tar.gz
sudo cp $RELEASE/aretext /usr/local/bin/
```

### macOS ARM 64-bit

```
VERSION=1.7.0
RELEASE=aretext_v${VERSION}_darwin_arm64
curl -LO https://github.com/aretext/aretext/releases/download/v$VERSION/$RELEASE.tar.gz
tar -zxvf $RELEASE.tar.gz
sudo cp $RELEASE/aretext /usr/local/bin/
```

Build From Source
-----------------

If you have [installed go](https://golang.org/doc/install), then you can build aretext from source:

```
mkdir -p $(go env GOPATH)/bin
git clone https://github.com/aretext/aretext.git
cd aretext
make install
```

This will install aretext in `$(go env GOPATH)/bin`, which you can add to your `$PATH` environment variable. If you use bash, put this line in your `~/.bashrc` or `~/.bash_profile`:

```
export PATH=$PATH:$(go env GOPATH)/bin
```

### Windows

There are no official Windows binaries yet, so build from source. In PowerShell:

```
git clone https://github.com/aretext/aretext.git
cd aretext
go install
```

This installs `aretext.exe` in `$(go env GOPATH)\bin`, which you can add to your `Path` environment variable.

Alternatively, build `aretext.exe` in the repository directory without installing it:

```
go build -o aretext.exe
```

Run aretext in [Windows Terminal](https://learn.microsoft.com/en-us/windows/terminal/), which supports the terminal features aretext needs. Shell commands run in PowerShell by default; see [Custom Menu Commands](custom-menu-commands.md) to change this.

Packages
--------

If a package is not yet available for your platform, please consider creating one! We are looking for package maintainers on Debian, Fedora, Homebrew, Nix, and any other platform you may prefer!

### Arch Linux

aretext is available as an [AUR Package](https://aur.archlinux.org/packages/aretext). If you use [yay](https://github.com/Jguer/yay), run this to install it:

```shell
yay -S aretext
```
