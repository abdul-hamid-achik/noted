# Installation

noted is distributed as a single static binary. Choose the method that fits your setup.

## Requirements

- macOS, Linux, or Windows
- A terminal with truecolor support recommended (the UI is optimized for Ghostty)

## Homebrew (recommended)

```bash
brew install abdul-hamid-achik/tap/noted
```

Upgrade later with:

```bash
brew upgrade noted
```

## Go install

Requires Go 1.25 or later:

```bash
go install github.com/abdul-hamid-achik/noted@latest
```

Make sure `$GOPATH/bin` (or `$(go env GOPATH)/bin`) is on your `PATH`.

## Download a release binary

Grab a pre-built binary from the [releases page](https://github.com/abdul-hamid-achik/noted/releases).

Available for:

- macOS (Intel and Apple Silicon)
- Linux (amd64 and arm64)
- Windows (amd64 and arm64)

## Build from source

```bash
git clone https://github.com/abdul-hamid-achik/noted.git
cd noted
task build
```

Plain `go build -o noted .` also works.
