![Logo](img/logo.svg)

# $ qrcp

Transfer files over Wi-Fi from your computer to a mobile device by scanning a QR code without leaving the terminal.

[![Go Report Card](https://goreportcard.com/badge/github.com/claudiodangelis/qrcp)](https://goreportcard.com/report/github.com/claudiodangelis/qrcp)

You can support development by donating with [![Buy Me A Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/claudiodangelis).

Join the **Telegram channel** [qrcp_dev](https://t.me/qrcp_dev) or the [@qrcp_dev](https://twitter.com/qrcp_dev) **Twitter account** for news about the development.

---

## How does it work?

![Screenshot](img/screenshot.png)

`qrcp` binds a web server to the address of your Wi-Fi network interface on a random port and creates a handler for it. The default handler serves the content and exits the program when the transfer is complete. When used to receive files, `qrcp` serves an upload page and handles the transfer.

The tool prints a QR code that encodes the text:

```
http://{address}:{port}/{random_path}
```

Most QR apps can detect URLs in decoded text and act accordingly (i.e., open the decoded URL with the default browser), so when the QR code is scanned, the content will begin downloading by the mobile browser.

### Demo

**Send files to mobile:**

![screenshot](img/demo.gif)

**Receive files from mobile:**

![Screenshot](img/mobile-demo.gif)

---

## Installation

### Using Go (Latest Development Version)
Requires Go 1.18 or later:
```sh
go install github.com/claudiodangelis/qrcp@latest
```

### Prebuilt Binaries
Download the latest release for your platform from the [Releases](https://github.com/claudiodangelis/qrcp/releases) page.

| Platform    | Instructions                                                                             |
|-------------|------------------------------------------------------------------------------------------|
| **Linux**   | Extract the `.tar.gz` archive, move the binary to `/usr/local/bin`, and set permissions. |
| **Windows** | Extract the `.tar.gz` archive and place the `.exe` file in a directory in your `PATH`.   |
| **macOS**   | Extract the `.tar.gz` archive, move the binary to `/usr/local/bin`, and set permissions. |

### Package Managers

| Platform    | Package Manager | Command                                        |
|-------------|-----------------|------------------------------------------------|
| **Linux**   | ArchLinux (AUR) | `yay -S qrcp-bin` or `yay -S qrcp`             |
| **Linux**   | Debian/Ubuntu   | `sudo dpkg -i qrcp_<version>_linux_x86_64.deb` |
| **Linux**   | CentOS/Fedora   | `sudo rpm -i qrcp_<version>_linux_x86_64.rpm`  |
| **Windows** | WinGet          | `winget install --id=claudiodangelis.qrcp  -e` |
| **Windows** | Scoop           | `scoop install qrcp`                           |
| **Windows** | Chocolatey      | `choco install qrcp`                           |
| **macOS**   | Homebrew        | `brew install qrcp`                            |

### Confirm Installation
After installation, verify that `qrcp` is working:
```sh
qrcp --help
```

---

## Usage

### Send Files

| Action                      | Command Example                   |
|-----------------------------|-----------------------------------|
| **Send a file**             | `qrcp MyDocument.pdf`             |
| **Send multiple files**     | `qrcp MyDocument.pdf IMG0001.jpg` |
| **Send a folder**           | `qrcp Documents/`                 |
| **Zip before transferring** | `qrcp --zip LongVideo.avi`        |

### Receive Files

| Action                              | Command Example                  |
|-------------------------------------|----------------------------------|
| **Receive to current directory**    | `qrcp receive`                   |
| **Receive to a specific directory** | `qrcp receive --output=/tmp/dir` |

---

## Configuration

`qrcp` works without prior configuration, but you can customize it using a configuration file or environment variables.

### Configuration File
The default configuration file is stored in `$XDG_CONFIG_HOME/qrcp/config.yml`. You can specify a custom location using the `--config` flag:
```sh
qrcp --config /tmp/qrcp.yml MyDocument.pdf
```

### Configuration Options

| Key         | Type    | Description                                                                    |
|-------------|---------|--------------------------------------------------------------------------------|
| `interface` | String  | Network interface to bind the web server to. Use `any` to bind to `0.0.0.0`.   |
| `bind`      | String  | Address to bind the web server to. Overrides `interface`.                      |
| `port`      | Integer | Port to use. Defaults to a random port.                                        |
| `path`      | String  | Path to use in the URL. Defaults to a random string.                           |
| `output`    | String  | Default directory to receive files. Defaults to the current working directory. |
| `fqdn`      | String  | Fully qualified domain name to use in the URL instead of the IP address.       |
| `keepAlive` | Bool    | Keep the server alive after transferring files. Defaults to `false`.           |
| `secure`    | Bool    | Use HTTPS instead of HTTP. Defaults to `false`.                                |
| `tls-cert`  | String  | Path to the TLS certificate. Used only when `secure: true`.                    |
| `tls-key`   | String  | Path to the TLS key. Used only when `secure: true`.                            |

### Environment Variables
All configuration parameters can also be set via environment variables prefixed with `QRCP_`:
- `$QRCP_INTERFACE`
- `$QRCP_PORT`
- `$QRCP_KEEPALIVE`

---

## Advanced Usage

### Network Interface
To use a specific network interface:
```sh
qrcp -i tun0 MyDocument.pdf
```

To bind the web server to all interfaces:
```sh
qrcp -i any MyDocument.pdf
```

### HTTPS
Enable secure transfers with HTTPS by providing a TLS certificate and key:
```sh
qrcp --tls-cert /path/to/cert.pem --tls-key /path/to/cert.key MyDocument.pdf
```

---

## Shell Completion

`qrcp` provides shell completion scripts for Bash, Zsh, and Fish.

| Shell    | Command Example                             |
|----------|---------------------------------------------|
| **Bash** | `source <(qrcp completion bash)`            |
| **Zsh**  | `qrcp completion zsh > "${fpath[1]}/_qrcp"` |
| **Fish** | `qrcp completion fish | source`             |

---

## Authors and Credits

- **Author**: [Claudio d'Angelis](https://t.me/claudiodangelis)
- **Logo**: Provided by [@arasatasaygin](https://github.com/arasatasaygin) as part of the [openlogos](https://github.com/arasatasaygin/openlogos) initiative.
- **Releases**: Managed with [goreleaser](https://goreleaser.com).

---

## Clones and Similar Projects

| Project Name                                                                                 | Description                                                                        |
|----------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------|
| [qr-fileshare](https://github.com/shivensinha4/qr-fileshare)                                 | A similar idea executed in NodeJS with a React interface.                          |
| [instant-file-transfer](https://github.com/maximumdata/instant-file-transfer) _(Uncredited)_ | Node.js project similar to this.                                                   |
| [qr-filetransfer](https://github.com/sdushantha/qr-filetransfer)                             | Python clone of this project.                                                      |
| [qr-filetransfer](https://github.com/svenkatreddy/qr-filetransfer)                           | Another Node.js clone of this project.                                             |
| [qr-transfer-node](https://github.com/codezoned/qr-transfer-node)                            | Another Node.js clone of this project.                                             |
| [QRDELIVER](https://github.com/realdennis/qrdeliver)                                         | Node.js project similar to this.                                                   |
| [qrfile](https://github.com/sgbj/qrfile)                                                     | Transfer files by scanning a QR code.                                              |
| [quick-transfer](https://github.com/CodeMan99/quick-transfer)                                | Node.js clone of this project.                                                     |
| [share-file-qr](https://github.com/pwalch/share-file-qr)                                     | Python re-implementation of this project.                                          |
| [share-files](https://github.com/antoaravinth/share-files) _(Uncredited)_                    | Yet another Node.js clone of this project.                                         |
| [ezshare](https://github.com/mifi/ezshare)                                                   | Another Node.js two-way file sharing tool supporting folders and multiple files.   |
| [local_file_share](https://github.com/woshimanong1990/local_file_share)                      | _"Share local file to other people, OR smartphone download files which is in PC."_ |
| [qrcp](https://github.com/pearl2201/qrcp)                                                    | A desktop app clone of `qrcp`, written with C# and .NET Core, works for Windows.   |
| [swift_file](https://github.com/mateoradman/swift_file)                                      | Rust project inspired by `qrcp`.                                                   |
| [qrcp-android](https://github.com/ianfixes/qrcp-android)                                     | Android app inspired by `qrcp`.                                                    |

---

## License

MIT. See [LICENSE](LICENSE).
