# Archon

Private server implementation for Phantasy Star Online Blue Burst by SEGA.

The goal of the Archon project is to build a modern, easy-to-use, customizable, and 
high-performing PSOBB server that can be run across multiple platforms with little 
setup overhead. The project is currently in development and changing rapidly while I 
piece together the PSO protocol and lean how the client works.

Credit is due to the authors of [Tethealla](http://pioneer2.net), 
[Sylverant](http://sylverant.net), and [Newserv](http://www.fuzziqersoftware.com), 
whose servers I'm studying as I write Archon.

Forks, bug fixes, issue reports, explanations of some of the client's bizarre behavior, 
etc. are more than welcome! I try to keep the development pretty open but if you have
questions feel free to open an issue.

* [Installation](#installation)
  + [Prerequisites:](#prerequisites-)
  + [1. Compile the code](#1-compile-the-code)
  + [2. Create a directory for the server files](#2-create-a-directory-for-the-server-files)
  + [3. Copy the supporting files](#3-copy-the-supporting-files)
  + [4. Create the database](#4-create-the-database)
  + [5. Set the hostname](#5-set-the-hostname)
  + [6. Point a PSOBB client at the server](#6-point-a-psobb-client-at-the-server)
  + [7. Add files the patch directory](#7-add-files-the-patch-directory)
  + [8. Run the server](#8-run-the-server)
* [Administration](#administration)
  + [Updating the server](#updating-the-server)
* [Contributing](#contributing)

## Installation

*Note*: The provided commands are aimed at MacOS/Linux but running their Windows
equivalents on a Windows system should still set the server up correctly.   

### Prerequisites:
* [Go](https://golang.org)
* [Git](https://git-scm.com/)
* [PostgreSQL](https://www.postgresql.org/)
* A C compiler for your system

### 1. Compile the code

Assuming Go installed:

    git clone https://github.com/dcrodman/archon.git
    cd archon
    mkdir .bin
    export GOBIN=$(pwd)/.bin
    go install ./cmd/*
    
This will install the Archon server and tools to the `.bin` subdirectory in the root 
of your project's directory.

### 2. Create a directory for the server files

This isn't necessary to run the server, however you may find it easier to have the 
server executable, tools, and supporting files all in once place. If you choose to 
go this route then from the directory in which you want the server files to reside:

    mkdir archon_server
    cp path-to-cloned-code/.bin/* .

In the following steps you'll need to update `config.yaml` with the full path to
any subdirectories you create (for instance, `patch_server.patch_dir`).

### 3. Copy the supporting files

Archon expects a few files in order to run. To

    cp -r path-to-cloned-code/setup/* .

The `setup/config.yaml` file contains all configuration options available to Archon, 
set to (hopefully) sane defaults.
   
Archon will also look for the config file in `/usr/local/etc/archon` if you're running
the server binary separately from the of the support files.

### 4. Create the database

Archon uses Postgres for persistent storage, which means you'll need to have a PostgreSQL
database instance running. Once you have one ready to go (assuming you have the Postgres
CLI tools available on your PATH): 

    createdb archondb
    psql archondb
    > CREATE USER archonadmin WITH ENCRYPTED PASSWORD 'psoadminpassword';
    > GRANT ALL ON ALL TABLES IN SCHEMA public TO archonadmin;

Feel free to choose your own credentials or database location, just make sure the settings in your 
`config.yaml` reflect them. Archon takes care of creating the tables and performing any migrations.
 
### 5. Set the hostname

In order for clients outside your network to connect, Archon needs to listen on a network interface. Once
you know your server's IP address, update `hostname` and `external_ip` in `config.yaml`. These values may
be the same but if the server will be running on a private subnet (like a home network) then `hostname` 
should be set to the IP assigned by the router and the `external_ip` to the internet-facing address.

Note: If the server will be hosted on a machine in a private network, you'll need to set up port forwarding
on the router between the server ports and the machine running Archon. 

### 6. Point a PSOBB client at the server

There are a few possible ways to accomplish this:  
  1. Update the connection addresses in the PSOBB client executable
  2. Override the psobb domains in users' hosts file
  3. Configure a DNS server that sends the psobb domains to your server

I may write a DNS server for this one day but for now option #1 is the simplest. You can either grab
a hex editor and change the addresses in the client yourself OR use the patcher utility that comes 
with Archon. To use the patcher (which should be in your server directory if you followed the optional)
step above:

    ./patcher -address <server-address> -exe <path-to-psobb-exe> 

A copy of the PSOBB client can be found here (as well as some additional instructions if they're helpful):
https://www.pioneer2.net/community/threads/tethealla-server-setup-instructions.1/

### 7. Add files the patch directory

It's recommended that you take the critical files from the copy of the client you intend for people to
use and put the majority of them in the patch directory (`patch_server.patch_dir` in the config file).
Archon will load these files and verify that they haven't been tampered with when the client connects,
which can help improve stability as well as make cheating harder.

    mkdir patches
    # copy your client files into ^

### 8. Run the server

The moment of truth; run the server by running this from your server directory:

    ./server

If everything's been configured correctly, you should get a bunch of messages about the different
sub-servers waiting for connections on the configured ports.

## Administration

### Updating the server

While individual commits may at points break `master`, the current HEAD of `master` should at all
times reference a fully functioning server. It should generally be safe to update your versiom by
doing the following:

    cd path-to-cloned-code
    git pull
    export GOBIN=$(pwd)/.bin
    go install ./cmd/*
    cp .bin/server your-server-directory

At the time of writing Archon doesn't yet have a recommended way of doing a no-downtime upgrade.
There are ways to mitigate this (like running a script to do this when nobody is connected) but
for now this is up to server admins to work out what works for them.

## Contributing

This is a pretty large project and pull requests, issues, and discussions are greatly appreciated!
If you'd like to get started contributing code to Archon, check out the 
[ Developer's Guide](https://github.com/dcrodman/archon/wiki/Developer's-Guide).
