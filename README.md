# wsstat [![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]

[license]: /LICENSE

The aim of this project is to provide a simple and easy to use tool to check the status of a WebSocket endpoint.

It is basically trying to do what [reorx/httpstat](https://github.com/reorx/httpstat) and [davecheney/httpstat](https://github.com/davecheney/httpstat) does for HTTP requests, but for WebSocket connections, and quite clearly draws a lot of inspiration from those two.

> Imitation is the sincerest form of flattery.

## Installation

There are a number of ways to install this tool, depending on your preference. All of them have the same result in common: you will be able to run `wsstat` from your terminal.

### Go installation

Requires that you have Go installed on your system and that you have `$GOPATH/bin` in your `PATH`. Recommended Go version is 1.21 or later.

Install via Go:

```sh
# To install the latest version, specify other releases with @<tag>
go install github.com/jakobilobi/wsstat@latest

# To include the version in the binary, run the install from the root of the repo
git clone github.com/jakobilobi/wsstat
cd wsstat
git fetch --all
git checkout origin/main
go install -ldflags "-X main.version=$(cat VERSION)" github.com/jakobilobi/wsstat@latest
```

Note: installing the package with `@latest`  will always install the latest version no matter the other parameters of the command.

### Binary download

#### Linux & macOS

Download the binary appropriate for your system from the latest release on [the release page](https://github.com/jakobilobi/wsstat/releases).

Make the binary executable:

```sh
chmod +x wsstat-<OS>-<ARCH>
```

Move the binary to a directory in your `PATH`:

```sh
sudo mv wsstat-<OS>-<ARCH> /usr/local/bin/wsstat  # system-wide
mv wsstat-<OS>-<ARCH> ~/bin/wsstat  # user-specific, ensure ~/bin is in your PATH
```

#### Windows

1. Download the `wsstat-windows-<ARCH>.exe` binary from the latest release on [the release page](https://github.com/jakobilobi/wsstat/releases).
2. Place the binary in a directory of your choice and add the directory to your `PATH` environment variable.
3. Rename the binary to `wsstat.exe` for convenience.
4. You can now run `wsstat` from the command prompt or PowerShell.

## Usage

Basic usage:

```sh
wsstat example.org
```

With verbose output:

```sh
wsstat -v ws://example.local
```

For more options:

```sh
wsstat -h
```
