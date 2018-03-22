# qr-filetransfer

Transfer files over wifi from your computer to your mobile device by scanning a QR code without leaving the terminal.

`<!-- TODO: insert GIF here -->`

![screenshot](screenshot.jpg)

## Install

```
go get github.com/claudiodangelis/qr-filetransfer
```

## How does it work?

This tool binds a web server to the address of your wifi network interface on a random port, and sets a default handler for it. The default handler serves the content and quits the program when the transfer is complete.

The program prints a QR code that encodes the text:

```
http://{address}:{port}
```

Most QR apps can detect URLs in decoded text and act accordingly (i.e.: open the URL with the default browser), so when QR the code is scanned the content starts being downloaded by the mobile browser.

## Usage

**Note**: Both computer and device must be on the same wifi network.

On its first run, `qr-filetransfer` will ask you to choose which **network interface** you want to use to transfer the files. Choose the network interface that is connected to your wifi:

```
$ qr-filetransfer /tmp/file
Choose the network interface to use (type the number):
[0] enp3s0
[1] wlp0s20u10
```

_Note: On Linux it usually starts with `wl`._

The choice will be remembered and you will never be prompted again, unless you pass the `-force` argument, or delete the `.qr-filetransfer.json` file that the program stores in the home directory of current user.



---


Transfer a single file

```
qr-filetransfer /path/to/file.txt
```

Zip the file, then transfer it

```
qr-filetransfer -zip /path/to/file.txt
```

Transfer a full directory. Note: the **directory gets zipped** before being transfered

```
qr-filetransfer /path/to/directory
```


## Arguments

- `-debug` increases verbosity
- `-force` ignores saved configuration
- `-zip` zips the content before transferring it


## Authors

- [Claudio d'Angelis](claudiodangelis@gmail.com) ([@daw985](https://twitter.com/daw985) on Twitter)

- [You?](https://github.com/claudiodangelis/qr-filetransfer/fork)
