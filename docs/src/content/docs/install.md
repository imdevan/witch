---
title: Installation
description: Installation instructions for witch
---


## Homebrew

```bash
brew install imdevan/witch/witch
```

## GitHub Release

Download the latest binary for your platform from the [releases page](https://github.com/imdevan/witch/releases).

```bash
# Linux (amd64)
curl -L https://github.com/imdevan/witch/releases/latest/download/witch-linux-amd64.tar.gz | tar -xz
sudo mv witch-linux-amd64 /usr/local/bin/witch
```

```bash
# macOS (Apple Silicon)
curl -L https://github.com/imdevan/witch/releases/latest/download/witch-darwin-arm64.tar.gz | tar -xz
sudo mv witch-darwin-arm64 /usr/local/bin/witch
```

```bash
# macOS (Intel)
curl -L https://github.com/imdevan/witch/releases/latest/download/witch-darwin-amd64.tar.gz | tar -xz
sudo mv witch-darwin-amd64 /usr/local/bin/witch
```

## Go

```bash
go install github.com/imdevan/witch/cmd/witch@latest
```

## Manual

```bash
gh repo clone imdevan/witch
cd witch
just build
sudo just install
```

## Next steps

Point witch at your dotfiles so it can find aliases and functions:

```bash
witch config init --editor
```

Then uncomment and edit the `[[sources]]` entries. See [CONFIG.md](CONFIG.md).

