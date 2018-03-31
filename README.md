# qr-filetransfer

Transfer files over Wi-Fi from your computer to a mobile device by scanning a QR code without leaving the terminal.

![screenshot](demo.gif)

## Install

```bash
go get github.com/claudiodangelis/qr-filetransfer
```

### Installation through a package manager

[AUR (Arch Linux)](https://aur.archlinux.org/packages/qr-filetransfer-git/)

## How does it work

This tool binds a web server to the address of your Wi-Fi network interface on a random port and creates a handler for it. The default handler serves the content and exits the program when the transfer is complete.

The tool prints a QR code that encodes the text:

```bash
http://{address}:{port}
```

Optionally, The file can be uploaded to a 3rd party site for hosting [file.io](https://file.io/).

Most QR apps can detect URLs in decoded text and act accordingly (i.e. open the decoded URL with the default browser), so when the QR code is scanned the content will begin downloading by the mobile browser.

## Usage

![Screenshot](screenshot.jpg)

**Note**: Both the computer and device must be on the same Wi-Fi network.

On its first run, `qr-filetransfer` will ask you to choose which **network interface** you want to use to transfer the files. Choose the network interface that is connected to your Wi-Fi:

```bash
$ qr-filetransfer /tmp/file
Choose the network interface to use (type the number):
[0] enp3s0
[1] wlp0s20u10
```

_Note: On Linux it usually starts with `wl`._

The chosen network will be saved and no more setup is necessary, unless you pass the `-force` argument, or delete the `.qr-filetransfer.json` file that the program stores in the home directory of current user.

---

Transfer a single file

```bash
qr-filetransfer /path/to/file.txt
```

Zip the file, then transfer it

```bash
qr-filetransfer -zip /path/to/file.txt
```

Transfer a full directory. Note: the **directory gets zipped** before being transferred

```bash
qr-filetransfer /path/to/directory
```

Use 3rd party file host ([file.io](https://file.io)) to serve the file instead of hosting it locally. Useful if your computer is not on the same wifi as your phone, or if your wifi separates wireless and LAN devices.

```bash
qr-filetransfer -remote /path/to/file.txt
```

## Arguments

- `-debug` increases verbosity
- `-quiet` ignores non critical output
- `-force` ignores saved configuration
- `-zip` zips the content before transferring it
- `-remote` uploads the file to [file.io](https://file.io) and shows a QR code to that URL

## Authors

- [Claudio d'Angelis](claudiodangelis@gmail.com) ([@daw985](https://twitter.com/daw985) on Twitter)

- [You?](https://github.com/claudiodangelis/qr-filetransfer/fork)
