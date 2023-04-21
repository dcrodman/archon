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

### Archon
The out-of-the-box configuration is geared towards development, so you should just be able to follow the basic README
instructions for installing and running it. Isn't that nice? I hope it's nice.

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
to overwrite the IP addresses to which the client will attempt to connect. Run the following from your Archon checkout
(replacing your IP and path):
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

As a side note, it may be helpful to assign your computer an IP address in the DHCP settings on your router so that
you don't have to re-patch the client whenever your computer gets a new subnet IP. I personally bind archon to my
Mac's IP on the Parallels bridge network so that it's always accessible from the VM and I don't have to futz with that.

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

You're off to the races. Feel free to open an issue with suggestions or comments on this guide if there's any
more detail that would be helpful (or anything is incorrect).
