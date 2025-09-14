🎮 Blue Archive Data Sync (ba-sync)

<div align="center">

<picture><img src="https://img.shields.io/badge/dynamic/yaml?url=https%3A%2F%2Fba.pokeguy.dev%2Fcom.nexon.bluearchive%2Fversion.txt&query=%24&prefix=v&style=for-the-badge&logo=nexon&label=Global&color=0099ff" alt="Nexon BlueArchive Latest Version"></picture>

<picture><img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"></picture>

</div>

## 🚀 Overview

`ba-sync` is a CLI tool that syncs Blue Archive client asset data to an Android device over ADB. It reads the device’s current preload metadata, computes the delta against the latest server resources, applies xdelta patches where possible, and downloads fresh files when needed. Caching is built-in for faster subsequent syncs.

> Note: At the moment, only the Global server and Preload data are supported.

### ✨ Key Features

- **📊 Version Awareness**: Checks device game version vs latest release and updates the APK/XAPK if needed
- **🔁 Incremental Sync**: Applies xdelta patches for changed files; downloads fresh files for new/missing ones
- **⚡ Concurrency**: Multi-threaded processing for faster syncs
- **🧠 Local Cache**: Stores processed files to avoid repeated work
- **🔍 Verbose Mode**: Optional detailed logs for diagnostics

## 📋 Prerequisites

| Tool | Minimum Version | Notes |
|:----:|:---------------:|:------|
| Go | ≥ 1.22 | For building/running the CLI |
| ADB | latest | Must be accessible on PATH |
| xdelta3 | latest | Must be accessible as `xdelta3` (or `xdelta`) on PATH |

Device requirements:
- The device must allow writing to `Android/data/com.nexon.bluearchive/files/PUB`. On many recent Android versions, this may require root or a build/environment that grants `adb shell` access to `Android/data`.
- Standard shell utilities available via toybox/busybox on the device: `stat`, `grep`, `cut`, `ls`, `mkdir`, `unzip`, `chown` (some operations may require root).

## ⚙️ Setup

### 1) Clone

```bash
git clone https://github.com/arisu-archive/bluearchive-data-sync.git
cd bluearchive-data-sync
```

### 2) Build or Run

```bash
# Build the tool (Makefile)
make build  # produces a binary named "sync"

# Or build with go directly (custom name)
go build -o ba-sync ./cmd/ba-sync

# Or run directly
go run ./cmd/ba-sync --help
```

## 🛠️ Usage

Top-level help:

```bash
ba-sync --help
```

Currently available command(s):
- `sync` — Sync asset data to the Android device

Global flags:
- `-v, --verbose` — Enable verbose logging

### `sync` command

Syncs preload assets to the connected Android device.

Flags:
- `-s, --serial <serial>`: Android device serial to target
- `-a, --host <ip:port>`: ADB TCP host (mutually exclusive with `--serial`)
- `-c, --cache-path <path>`: Cache directory (default: OS cache dir `/.bluearchive-data-sync`)
- `-r, --server <server>`: Resource server (default: `global`; currently only `global` supported)
- `--preload`: Only sync preload data (default: `true`; only preload is supported)
- `--forced`: Force reprocess all files (ignore device hashes)
- `--concurrency <N>`: Number of concurrent workers (default: 16)

Examples:

```bash
# Sync using a USB device by serial
ba-sync sync --serial R3CN90ABCDE

# Sync using a network-connected ADB device
ba-sync sync --host 192.168.1.50:5555

# Force reprocess all files with higher concurrency and verbose logging
ba-sync sync -s R3CN90ABCDE --forced --concurrency 32 -v

# Specify a custom cache directory
ba-sync sync -s R3CN90ABCDE --cache-path "C:/Users/you/.ba-sync-cache"
```

Exit codes: returns non-zero on failure; see logs for details.

## 🔍 How It Works

1. **Device Detection**: Connects via ADB using `--serial` or `--host`.
2. **Compatibility Check**: Verifies the Blue Archive app is installed; updates it if a newer version is available.
3. **State Discovery**: Reads device `patch.version.map` and `patch.file.hash` from `Android/data/com.nexon.bluearchive/files/PUB/Patch`.
4. **Analysis**: Compares device hashes with latest server resources, marking files as Patch/New/Skip.
5. **Processing**: Applies xdelta patches for changed files; downloads full files for new/missing ones, with concurrency.
6. **Deployment**: Pushes processed files into `Android/data/com.nexon.bluearchive/files/PUB/Resource/...` and fixes ownership.
7. **Metadata Update**: Updates device `patch.version.map` and `patch.file.hash` to reflect the new state.

## 🧪 Troubleshooting

- **xdelta3 not found**: Ensure `xdelta3` (or `xdelta`) is installed and on PATH.
- **Permission denied under Android/data**: Your device may require root or special permissions to write under `Android/data`. Verify your environment permits `adb shell` to write there.
- **Device not found**: Confirm `adb devices` lists your device, or connect over TCP with `adb connect <ip:port>` and then use `--host`.
- **Server not supported**: Only `global` is supported right now. Using other values will fail.
- **unzip not found on device**: The tool installs or updates the app by uploading an XAPK and running `unzip` on-device. Ensure your environment provides `unzip` (via toybox/busybox) or install it.

## 📈 Roadmap

- [ ] Japan server support
- [ ] Full data sync (beyond Preload)
- [ ] Improved permission handling for modern Android
- [ ] More granular selection/filtering

## 🤝 Contributing

Contributions are welcome!
1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m "Add amazing feature"`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

MIT — see `LICENSE`.

## ⚠️ Disclaimer

This tool is intended for research and analysis. Ensure compliance with the game’s terms of service, local laws, and regulations.

## 📬 Contact & Support

- Open an Issue for bugs or feature requests
- Star ⭐ the repo if you find it helpful
