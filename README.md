
![Logo](logo.svg)

# qrcp

Transfer files over Wi-Fi from your computer to a mobile device by scanning a QR code without leaving the terminal.

[![Go Report Card](https://goreportcard.com/badge/github.com/claudiodangelis/qrcp)](https://goreportcard.com/report/github.com/claudiodangelis/qrcp)

[![Buy Me A Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/claudiodangelis)


### Desktop to mobile

![screenshot](demo.gif)

### Mobile to desktop

![Screenshot](mobile-demo.gif)

## Install


### Installation with Go 

_Note: Go 1.8 is required to run._

```
go get github.com/claudiodangelis/qrcp
```

### Installation through a package manager

[AUR (Arch Linux)](https://aur.archlinux.org/packages/qrcp-git/)

## How does it work?


`qrcp` binds a web server to the address of your Wi-Fi network interface on a random port and creates a handler for it. The default handler serves the content and exits the program when the transfer is complete.

The tool prints a QR code that encodes the text:

```
http://{address}:{port}/{random_path}
```


Most QR apps can detect URLs in decoded text and act accordingly (i.e. open the decoded URL with the default browser), so when the QR code is scanned the content will begin downloading by the mobile browser.

## Usage


![Screenshot](screenshot.jpg)


**Note**: Both the computer and device must be on the same Wi-Fi network.

On the first run, `qrcp` will ask to choose which **network interface** to use to transfer the files. Note that if only one suitable network interface is found, it will be used without asking.

Two transfers mode are supported: **desktop-to-mobile** and **mobile-to-desktop**


## Desktop to Mobile

Transfer a single file

```
qrcp /path/to/file.txt
```

The `send` command is an alias created for consistency with the `receive` command described later on.
```sh
# These two commands are equivalent
qrcp send /path/to/file.txt
qrcp /path/to/file.txt
```

Zip the file, then transfer it

```
qrcp --zip /path/to/file.txt
```

Transfer a full directory. Note: the **directory is zipped** by the program before being transferred

```
qrcp /path/to/directory
```

## Mobile to Desktop

If you want to use it the other way, you need to use the `receive` command. By scanning the QR code you will be redirected to an "Upload" page where you can choose which file(s) you want to transfer.

By default the file(s) are transferred to the current directory, you can override that by passing the `--output` flag:

```sh
# This downloads the file(s) on the current directory
qrcp receive
```

```sh
# This downloads the file(s) in the ~/Downloads directory
qrcp receive --output ~/Downloads
```

If you don't want the web server to listen to a random port, you can specify one:

```sh
qrcp --port=8080 /path/to/my-file
```


## Tips & Tricks

### Keep server alive

If you are trying to transfer a file that the browser on the receiving end is considering harmful, you can be asked by the browser if you really want to keep the file or discard it; this condition (browser awaiting your answer) can lead to qrcp disconnection. To prevent qrcp from disconnecting, use the `--keep-alive` flag:

```sh
qrcp --keep-alive /path/to/my/totally/cool.apk
```


## Authors

**qrcp**, originally called **qr-filetransfer**, started from an idea of [Claudio d'Angelis](claudiodangelis@gmail.com) ([@daw985](https://twitter.com/daw985) on Twitter), the current maintainer, and it's [developed by the community](https://github.com/claudiodangelis/qrcp/graphs/contributors).


[Join us!](https://github.com/claudiodangelis/qrcp/fork)

## Logo Credits

Logo is provided by [@arasatasaygin](https://github.com/arasatasaygin) as part of the [openlogos](https://github.com/arasatasaygin/openlogos) initiative, a collection of free logos for open source projects.

Check out the rules to claim one: [rules of openlogos](https://github.com/arasatasaygin/openlogos#rules).

## Clones and Similar Projects

- [qr-fileshare](https://github.com/shivensinha4/qr-fileshare) - A similar idea executed in NodeJS with a React interface.
- [instant-file-transfer](https://github.com/maximumdata/instant-file-transfer) _(Uncredited)_ - Node.js project similar to this
- [qr-filetransfer](https://github.com/sdushantha/qr-filetransfer) - Python clone of this project
- [qr-filetransfer](https://github.com/svenkatreddy/qr-filetransfer) - Another Node.js clone of this project
- [qr-transfer-node](https://github.com/codezoned/qr-transfer-node) - Another Node.js clone of this project
- [QRDELIVER](https://github.com/realdennis/qrdeliver) - Node.js project similar to this
- [qrfile](https://github.com/sgbj/qrfile) - Transfer files by scanning a QR code
- [quick-transfer](https://github.com/CodeMan99/quick-transfer) - Node.js clone of this project
- [share-file-qr](https://github.com/pwalch/share-file-qr) - Python re-implementation of this project
- [share-files](https://github.com/antoaravinth/share-files) _(Uncredited)_  - Yet another Node.js clone of this project
- [ezshare](https://github.com/mifi/ezshare) - Another Node.js two way file sharing tool supporting folders and multiple files

## License

MIT. See [LICENSE](LICENSE).
