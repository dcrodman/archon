# Contributing

My overall strategy for the server has evolved over time but the general approach is to try to get as many packets at
least loosely implemented as possible. The development cycle usually goes something like this:
  1. Run a PSOBB client against Archon
  2. Client halts, crashes, or generally does something unexpected
  3. Figure out what packet Archon is missing by comparing to logs from Tethealla
  5. Implement whatever is required to properly send that packet (either by making my own guesses or by reading 
  Tethealla/Sylverant/Newserv to figure out what data is in there)
  6. Commit the code to Archon after testing all of it
  7. Repeat

Given the goal of just building in the basic support for as many packets as possible, I semi-often skim over things or
leave chunks of packets unused. Sometimes this is because the other servers do it, other times because I'm just punting
it for later. This is probably the biggest help for others to come in and fill in since it's easier to carve out a small
chunk and dive deep into what the other servers do.

## Getting Started
This will walk you through setting up Archon for development as well as a reference PSOBB server implementation. The
following sections assume you're using macOS or Linux, though if you're on Windows then the only tweaks you should need
to make are to some of the example paths. Either way you are going to need access to a Windows machine for running PSOBB
and (likely) working with Tethealla. 

It's possible to run the PSOBB client in [Wine](https://www.winehq.org/), though your mileage may vary. It doesn't work
particularly well with Visual Studio, which can be problematic if you find yourself needing to make changes to a copy of
vanilla Tethealla.

### PSO Clients
To do basically anything with this project you're going to need a PSOBB client. Conveniently, you can find a compatible 
client on [pioneer2.net](https://www.pioneer2.net/community/threads/full-client-download.41/). Download and extract it, 
which will leave you with the Tethealla v12513 version.

You may not need this step, however if you're running Archon on a different system than you're running PSO (for instance
Archon on a Mac and PSO on a Windows virtual machine), it's easiest to connect over the local network. By default, Archon
binds to `0.0.0.0` (all network interfaces) so all you need to do is drop the IP address of your Archon machine into a
client executable.

In the directory where you just extracted the PSO client, make a copy of `Psobb.exe`:
```
cp Psobb.exe Psobb_Archon.exe
```
Now we need to patch this executable to point to Archon. This project contains a tool in `cmd/patcher` that can be used
to overwrite the IP addresses to which the client will attempt to connect. As for running this tool, you can either
(depending on your virtualization software/settings):
1. Run `bin/patcher /path/to/your/windows/checkout` from the clone of Archon on your Mac. For example, this might be your path:
```
❯ ls /Volumes/\[C\]\ Windows\ 11.hidden/Users/dcrodman/PSO
GameGuard             Psobb Localhost.exe   bmp                   install.reg           option.exe
PSO.ini               Psobb.exe             data                  log                   teamflag
Psobb 192.168.1.5.exe Readme.txt            dummy.txt             online.exe            uninst
```
2. Clone Archon and build the executables on your Windows machine directly.

Either way, run the following from your Archon checkout (replacing your IP and path):
```
bin/patcher -address 192.168.1.5 /Volumes/\[C\]\ Windows\ 11.hidden/Users/dcrodman/PSO/Psobb_Archon.exe
patching exe with new address: 192.168.1.5
replacing 127.0.0.1 with 192.168.1.5
replacing 127.0.0.1 with 192.168.1.5
replacing 127.0.0.1 with 192.168.1.5
replacing 127.0.0.1 with 192.168.1.5
replacing 127.0.0.1 with 192.168.1.5
```
You now have a copy of a PSOBB client that can connect to Archon. 

As a side note, it will be helpful to assign your computer an IP address in the DHCP settings on your router so that
you don't have to re-patch the client whenever your computer gets a new subnet IP.

### Archon

Prerequisites:
* A working Postgres database

First, you'll need to clone the project (if you haven't already):
```
git clone https://github.com/dcrodman/archon.git && cd archon
```
Create the config directory, which we'll keep separately from the rest of the codebase.
```
mkdir /usr/local/etc/archon
cp -r setup/* /usr/local/etc/archon/
```
Update `/usr/local/etc/archon/config.yaml` if needed (for instance if you're setting this up on another machine). These
are likely the only configs you may want to modify:
```
# Hostname or IP address on which the servers will listen for connections.
hostname: 0.0.0.0
# IP broadcast to clients in the redirect packets.
external_ip: 127.0.0.1

database:
  # Hostname of the Postgres database instance.
  host: 127.0.0.1
  # Name of the database in Postgres for archon.
  name: archondb
  # Username and password of a user with full RW privileges to ${db_name}.
  username: archonadmin
  password: psoadminpassword
```
Create a Postgres database with credentials that match `config.yaml`:
```
$ createdb archondb
$ psql -d archondb -c "CREATE USER archonadmin WITH ENCRYPTED PASSWORD 'psoadminpassword'"
CREATE ROLE
$ psql -d archondb -c "GRANT ALL ON ALL TABLES IN SCHEMA public TO archonadmin"
GRANT
```
Build the binaries and run the tests/linting tools:
```
$ make
mkdir -p bin
go build -o bin ./cmd/*
golangci-lint run
go test ./...
?   	github.com/dcrodman/archon/cmd/account	[no test files]
?   	github.com/dcrodman/archon/cmd/certgen	[no test files]
?   	github.com/dcrodman/archon/cmd/patcher	[no test files]
?   	github.com/dcrodman/archon/cmd/server	[no test files]
?   	github.com/dcrodman/archon/cmd/sniffer	[no test files]
?   	github.com/dcrodman/archon/internal	[no test files]
?   	github.com/dcrodman/archon/internal/block	[no test files]
ok  	github.com/dcrodman/archon/internal/character	0.088s
?   	github.com/dcrodman/archon/internal/core	[no test files]
?   	github.com/dcrodman/archon/internal/core/bytes	[no test files]
?   	github.com/dcrodman/archon/internal/core/client	[no test files]
?   	github.com/dcrodman/archon/internal/core/data	[no test files]
?   	github.com/dcrodman/archon/internal/core/debug	[no test files]
ok  	github.com/dcrodman/archon/internal/core/encryption	0.115s
?   	github.com/dcrodman/archon/internal/core/proto	[no test files]
ok  	github.com/dcrodman/archon/internal/core/prs	0.083s
?   	github.com/dcrodman/archon/internal/login	[no test files]
?   	github.com/dcrodman/archon/internal/packets	[no test files]
?   	github.com/dcrodman/archon/internal/patch	[no test files]
?   	github.com/dcrodman/archon/internal/ship	[no test files]
?   	github.com/dcrodman/archon/internal/shipgate	[no test files]
```
You should now be able to run the server:
```
$ bin/server -config /usr/local/etc/archon
Archon PSO Backend, Copyright (C) 2014 Andrew Rodman
=====================================================
This program is free software: you can redistribute it and/or
modify it under the terms of the GNU General Public License as
published by the Free Software Foundation, either version 3 of
the License, or (at your option) any later version. This program
is distributed WITHOUT ANY WARRANTY; See LICENSE for details.
using configuration file: /usr/local/etc/archon
INFO[2022-10-31 20:17:38] starting pprof server on localhost:4000
INFO[2022-10-31 20:17:38] [SHIPGATE] connected to database 127.0.0.1:5432
2022/10/31 20:17:39 http: TLS handshake error from 127.0.0.1:61940: EOF
INFO[2022-10-31 20:17:39] [PATCH] waiting for connections on 192.168.1.5:11000
INFO[2022-10-31 20:17:39] loading patch files from /usr/local/etc/archon/patches
INFO[2022-10-31 20:17:39] [DATA] waiting for connections on 192.168.1.5:11001
INFO[2022-10-31 20:17:39] [LOGIN] waiting for connections on 192.168.1.5:12000
INFO[2022-10-31 20:17:39] loading parameters from /usr/local/etc/archon/parameters
INFO[2022-10-31 20:17:39] ItemMagEdit.prs (802 bytes, checksum: 0xa2bb2a80)
INFO[2022-10-31 20:17:39] ItemPMT.prs (21757 bytes, checksum: 0x2c0116de)
INFO[2022-10-31 20:17:39] BattleParamEntry.dat (62976 bytes, checksum: 0xfb6b06a8)
INFO[2022-10-31 20:17:39] BattleParamEntry_on.dat (62976 bytes, checksum: 0xd1f1f152)
INFO[2022-10-31 20:17:39] BattleParamEntry_lab.dat (62976 bytes, checksum: 0x8d628e0c)
INFO[2022-10-31 20:17:39] BattleParamEntry_lab_on.dat (62976 bytes, checksum: 0x48f1e518)
INFO[2022-10-31 20:17:39] BattleParamEntry_ep4.dat (62976 bytes, checksum: 0xe85bfab4)
INFO[2022-10-31 20:17:39] BattleParamEntry_ep4_on.dat (62976 bytes, checksum: 0x2eea31b)
INFO[2022-10-31 20:17:39] PlyLevelTbl.prs (11790 bytes, checksum: 0xc6652d45)
INFO[2022-10-31 20:17:39] [CHARACTER] waiting for connections on 192.168.1.5:12001
INFO[2022-10-31 20:17:39] [SHIPGATE] registered ship Default at 192.168.1.5:15000
INFO[2022-10-31 20:17:39] [SHIP] waiting for connections on 192.168.1.5:15000
INFO[2022-10-31 20:17:39] [BLOCK01] waiting for connections on 192.168.1.5:15002
INFO[2022-10-31 20:17:39] [BLOCK02] waiting for connections on 192.168.1.5:15003
```
The last thing to do is to set up a user account. Name it whatever you like:
```
$ bin/account -config /usr/local/etc/archon add
Username: test
Password: test
Email: a@b.c
``` 
Run the `Psobb_Archon.exe` client we set up earlier, drop in those credentials, and you're ready to go.

Note: If you're using VSCode, it's convenient to be able to run/debug the server from the editor:
```
// launch.json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "server/main.go",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/server",
            "args": [
                "-config",
                "/usr/local/etc/archon",
            ]
        }
    ]
}
```

### Reference Server

There are many approaches you can adopt to working out this protocol and I recommend that
you experiment and do whatever makes sense for you. However since the official servers have
long been decommissioned, as I see it there are only a few options:
1. Reverse engineer the client
2. Compare the packets sent to an existing server

As someone who is not a reverse engineering expert and only gets a few hours a week to hack 
on this, I've personally opted for the second approach. In order to do the same, you'll need
some combination of a:
* Copy of Tethealla
* (Optional) C debugger (included with Visual Studio, which you'll need if you modify Tethealla)
* (Optional) Packet sniffer

I typically use a set of captured packet data from various Client scenarios and do my best to work 
out what's going on, sometimes dipping into the Tethealla code or attaching a debugger to a running server
and setting breakpoints. Using the Visual Studio debugger is beyond the scope of this tutorial, however if
you need help with that or testing changes then drop into the Discord (linked in  the Readme) and I'll 
do my best to help.

First off, I recommend following [Sodaboy's official instructions](https://www.pioneer2.net/community/threads/tethealla-server-setup-instructions.1/)
for setting up a copy of Tethealla since this is (at the time of writing) the most functional open source
Blue Burst server.

The next thing you'll need is a clone of Archon on your Windows machine in order to build the client-side
packet sniffer. This requires installing:
* [Go](https://go.dev/dl/)
* [Npcap](https://npcap.com/)

Once those are ready, build the sniffer from your archon directory:
```
go build -o bin .\cmd\sniffer

# Note: If you're running Windows on an ARM machine (like an M1 mac),
# you'll need to cross-compile it instead:
env GOOS=windows GOARCH=386 go build -o bin .\cmd\sniffer
```

Assuming you've gotten all of that ready and the Tethealla servers are running, capturing data is easy. If
you're running tethealla on a different host then you'll need to find the correct network interface to listen
on. Passing the `-l` flag to the sniffer will list the available devices. 
```
Usage of bin/sniffer:
  -d string
    	Device on which to listen for packets (default "en0")
  -f string
    	BPF packet filter to apply (default "tcp portrange 11000-11002 or tcp portrange 12001-12003 or tcp portrange 15000-15003 or tcp portrange 5278-5290")
  -l	List devices
  -o string
    	File to which to output logs (default stdout)
```

If this is all just on localhost then this will initialize the sniffer to look for local traffic:
```
.\bin\sniffer.exe -d \Device\NPF_Loopback -o my_first_capture.txt
```

Open `Psobb.exe`, connect with whatever credentials you set up, and tinker around. Once you're done, Ctrl-C the
sniffer and your `my_first_capture.txt` should contain something like this:
```
❯ head -n 20 01_simple_to_lobby.log
[PATCH] 0x02 (PatchWelcomeType) | server->client (76 bytes)
(0000) 4c 00 02 00 50 61 74 63 68 20 53 65 72 76 65 72     L...Patch Server
(0010) 2e 20 43 6f 70 79 72 69 67 68 74 20 53 6f 6e 69     . Copyright Soni
(0020) 63 54 65 61 6d 2c 20 4c 54 44 2e 20 32 30 30 31     cTeam, LTD. 2001
(0030) 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00     ................
(0040) 00 00 00 00 a0 1b 07 59 8e 22 be 0f                 .......Y."¾.

[PATCH] 0x02 (PatchWelcomeType) | client->server (4 bytes)
(0000) 04 00 02 00                                           ....

[PATCH] 0x04 (PatchHandshakeType) | server->client (4 bytes)
(0000) 04 00 04 00                                           ....

[PATCH] 0x04 (PatchHandshakeType) | client->server (112 bytes)
(0000) 70 00 04 00 00 00 00 00 00 00 00 00 00 00 00 00     p...............
(0010) 74 65 73 74 00 00 00 00 00 00 00 00 00 00 00 00     test............
(0020) 74 65 73 74 00 00 00 00 00 00 00 00 00 00 00 00     test............
(0030) 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00     ................
(0040) 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00     ................
(0050) 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00     ................
(0060) 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00     ................
```

It can be really helpful to use a diff tool like difftastic to compare the differences between a run against
Tethealla (the "expected" output) and a run against Archon.

You're off to the races. Feel free to open an issue with suggestions or comments on this guide if there's any
more detail that would be helpful (or anything is incorrect).
