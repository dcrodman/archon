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
* [Installation](#Installation)
* [Setup Script](#setup-script)
  + [Script Prerequisites](#script-prerequisites)
* [Manual installation](#manual-installation)
  + [Manual Prerequisites](#manual-prerequisites)
  + [1. Compile the code](#1-compile-the-code)
  + [2. Create a directory for the server files](#2-create-a-directory-for-the-server-files)
  + [3. Copy the supporting files](#3-copy-the-supporting-files)
  + [4. Create the database](#4-create-the-database)
  + [5. Set the hostname](#5-set-the-hostname)
  + [6. Point a PSOBB client at the server](#6-point-a-psobb-client-at-the-server)
  + [7. Add files to the patch directory](#7-add-files-to-the-patch-directory)
  + [8. Generate the shipgate SSL certificates](#8-generate-the-shipgate-ssl-certificates)
  + [9. Add the first player account](#9-add-the-first-player-account)
  + [10. Run the server](#10-run-the-server)
* [Run in docker](#run-in-docker)
  + [Docker Prerequisites](#docker-prerequisites)
  + [How-to](#how-to)  
* [Administration](#administration)
  + [Updating the server](#updating-the-server)
* [Contributing](#contributing)

## Installation

There are three ways to run the existing server:
- manually build the server
- running the server in docker
- using the setup script

Consider using manual installation, or the setup script if you want to have full control on the tools and server 
you're running or if you want to change something on the fly.

If you mostly need to run an existing version, want to test it out or just want to use less 
tools - consider using dockerized setup.

## Setup Script

### Script Prerequisites

* [Go](https://golang.org)
* [Git](https://git-scm.com/)
* [PostgreSQL](https://www.postgresql.org/)

### 1. Clone Archon

    git clone https://github.com/dcrodman/archon.git

### 2. Run the setup script

The setup script is located in the `setup` directory. To run the script, execute the following with an optional
parameter specifying the install path:

    path-to-cloned-code/setup/setup.sh [install-path]

If the last directory in the install path does not exist, it will be created. 
If the install-path is omitted, the setup script will install to:

    path-to-cloned-code/archon_server

### 3. Follow the prompts

The script will guide you through the initial server configuration as well as prompt for credentials for the first PSO
account. Once the setup is complete, the script will provide additional configuration scripts, and a command to run the
server.

## Manual Installation

**Note**: The provided commands are aimed at MacOS/Linux but running their Windows
equivalents on a Windows system should still set the server up correctly.   

### Manual Prerequisites:
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
    cd archon_server
    cp path-to-cloned-code/.bin/* .

In the following steps you'll need to update `config.yaml` with the full path to
any subdirectories you create (for instance, `patch_server.patch_dir`).

**Note**: For the remainder of this guide, the commands assume that your current working
directory is the server directory you've just created. 

### 3. Copy the supporting files

Archon expects a few files in order to run, which can be retrieved from the setip directory:

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

**Note**: If you use a client other than the TethVer12513 executables, you may need to uncomment lines
in `patcher.go` that correspond to your client. If none exist, you'll have to find the offsets with a
hex editor.

### 7. Add files to the patch directory

It's recommended that you take the critical files from the copy of the client you intend for people to
use and put the majority of them in the patch directory (`patch_server.patch_dir` in the config file).
Archon will load these files and verify that they haven't been tampered with when the client connects,
which can help improve stability as well as make cheating harder.

    mkdir patches
    # copy your client files into ^

### 8. Generate the shipgate SSL certificates

The shipgate API server requires clients to connect over SSL as both a form of security as well as
mutual authentication. Archon includes a tool for generating these certificates, which need to be
present in the server's config directory:

    ./generate_cert

The tool will prompt you for your server's external_ip (which should be the same as `external_ip`
in `config.yaml`). You may also provide a CIDR block.

### 9. Add the first player account

You can do this with your own tool (or SQL) Archon comes with a small utility for adding accounts:

    ./account -config /path/to/config add

### 10. Run the server

The moment of truth; run the server by running this from your server directory:

    ./server -config /path/to/config

If everything's been configured correctly, you should get a bunch of messages about the different
sub-servers waiting for connections on the configured ports.

## Run in docker

### Docker Prerequisites:
* [Docker](https://www.docker.com)
* Assumes a recent docker version bundled together with **docker-compose** - otherwise compose should be installed too

### How-to

Change your working directory to `build` e.g. run `cd build`.

Run with `docker-compose up` - it will download required images and run both Postgres DB and the server.

There are 3 services available in current docker-compose version:
- `postgres` - PostgreSQL database with initial DB and tables created via script
- `account` - account tool which creates initial account for login (can be disabled or commented out if not needed)
- `server` - actual server running on 127.0.0.1 with PSO ports exposed

In dockerized setup server is running same commands as in the manual setup so it contains 
all the tools bundled in the container.

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
