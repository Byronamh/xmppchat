# xmppchat
A CLI based chat client using xmpp as it's protocol written in Go

## Installation

You need to have [go](https://golang.org/dl/) installed

Once you have golang installed, run the following commands
```shell
git clone https://github.com/Byronamh/xmppchat
cd xmppchat
go mod download
```

## Running the program

After dependencies have been installed, run the follow command to compile and run the program
```shell
go run src/main.go
```

## Program manual

At first, you will be asked to input your credentials, the following are required:
 - username (jid)
 - password
 - server host

After you input the parameters mentioned above, a connection attempt will be made on the server you provided. You will be asked to crete a user in the server in case it doesnt exist on the server.

Following regiostration/login the user will be shown the following menu. Incoming messages will be shown on screen.
you can type in `-help` to show the menu again. 