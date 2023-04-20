# Archon
![build](https://github.com/dcrodman/archon/actions/workflows/build.yml/badge.svg?branch=master) 
![discord](https://img.shields.io/discord/819749462468984923) 
![license](https://img.shields.io/github/license/dcrodman/archon) 

Private server implementation for Phantasy Star Online Blue Burst by SEGA.

The goal of the Archon project is to build a modern, easy-to-use, customizable, and 
high-performing PSOBB server that can be run across multiple platforms with little 
setup overhead. The project is currently in relatively active development and things
change frequently while I piece together the PSO protocol and lean how the client works.

Credit is due to the authors of [Tethealla](http://pioneer2.net), 
[Sylverant](http://sylverant.net), and [Newserv](http://www.fuzziqersoftware.com), 
whose servers I'm studying as I write Archon.

This is a long running project that I work on when I have time, which is pretty sporadic
given how time-intensive this endeavor is. That said, forks, bug fixes, issue reports,
explanations of some of the client's bizarre behavior, questions, etc. are welcome to
help move things along. Some starter information can be found in CONTRIBUTING.md.

## Running the server

Archon can be run out of the box with little to no configuration. It includes a starter
configuration file, the initialization files for the game, and uses SQLite by default. The
only two requirements are [Git](https://git-scm.com/) and [Go](https://go.dev/doc/install).

    git clone https://github.com/dcrodman/archon.git && cd archon
    make && bin/sever

Archon comes with a small utility for managing accounts, unless you want to run the SQL yourself. To add
player accounts, just run the tool and follow the prompts:

    bin/account add

### Overriding configs

The default configuration should be fine for most cases, though if you want other players to be able
to connect there are a couple of configs you'll need to change. The default configuration can be overridden
one of two ways:

1. Pass the `-config` flag to `bin/server` with a path to a directory containing the config file. This also
alters where the database will be created.
```
# if your config file is in /usr/local/etc/archon/config.yaml:
bin/server -config /usr/local/etc/archon
```
2. Set `ARCHON_` prefixed environment variables, for example `ARCHON_HOSTNAME` or `ARCHON_DATABASE_ENGINE`.

All configuration are defined and commented in `config.yaml`. Feel free to copy this file as a starting point.

### Hostname and Broadcast IP

In order for clients outside your network to connect, Archon needs to listen on an external network interface.
Once you know your server's IP address, update `hostname` and `external_ip` in `config.yaml`. These values may
be the same but if the server will be running on a private subnet (like a home network) then `hostname` 
should be set to the IP assigned by the router and the `external_ip` to the internet-facing address.

Note: If the server will be hosted on a machine in a private network, you'll need to set up port forwarding
on the router between the server ports and the machine running Archon. Even then this is only recommended
if you have a static IP; you're better off hosting this on a cloud server somewhere for actual gameplay.

### Add files to the patch directory

It's recommended that you take the critical files from the copy of the client you intend for people to
use and put the majority of them in the patch directory (`patch_server.patch_dir` in the config file).
Archon will load these files and verify that they haven't been tampered with when the client connects,
which can help improve stability as well as make cheating harder.

### Changing the database

Archon uses SQLite by default, but can easily be switched to use a [PostgreSQL](https://www.postgresql.org/) 
database (or others, PRs welcome) if you prefer. Assuming a working Postgres installation, you need only add
or uncomment and the following lines in the config file:

    database:
        engine: postgres
        host: 127.0.0.1
        port: 5432
        name: archondb
        username: archonadmin
        password: psoadminpassword
        ## Set to verify-full if the Postgres instance supports SSL.
        sslmode: disable

then create the database (substitute the credentials if you wish, they just have to match the config file):

    createdb archondb
    psql archondb
    > CREATE USER archonadmin WITH ENCRYPTED PASSWORD 'psoadminpassword';
    > GRANT ALL ON ALL TABLES IN SCHEMA public TO archonadmin;

## Connecting clients

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
