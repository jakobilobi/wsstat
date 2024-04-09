# wsstat [![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]

[license]: /LICENSE

The aim of this project is to provide a simple and easy to use tool to check the status of a WebSocket endpoint.

It is basically trying to do what [reorx/httpstat](https://github.com/reorx/httpstat) and [davecheney/httpstat](https://github.com/davecheney/httpstat) does for HTTP requests, but for WebSocket connections, and quite clearly draws a lot of inspiration from those two.

> Imitation is the sincerest form of flattery.

## Installation

Recommended Go version is 1.21 or later.

Install via Go:

    go install github.com/jakobilobi/wsstat@latest

TODO: add binary download install option

## Usage

Basic usage:

```sh
wsstat wss://example.org
```

With verbose output:

```sh
wsstat -v wss://example.org
```

For more options:

```sh
wsstat -h
```
