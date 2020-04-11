# Archon

Private server implementation for Phantasy Star Online Blue Burst by SEGA.

The goal of the Archon project is to build a modern, easy-to-use, customizable, and 
high-performing PSOBB server that can be run across multiple platforms with little 
setup overhead. The project is currently in development and changing rapidly while I 
piece together the PSO protocol and develop a core architecture.

Credit is due to the authors of [Tethealla](http://pioneer2.net), 
[Sylverant](http://sylverant.net), and [Newserv](http://www.fuzziqersoftware.com), 
whose servers I'm studying as I write Archon.

Forks, bug fixes, issue reports, explanations of some of the client's bizarre behavior, 
etc. are more than welcome! I try to keep the development pretty open but if you have
questions feel free to open an issue.

- [Installation](#installation)
    * [1. Compile the code](#1-compile-the-code)
    * [2. Set up the config file](#2-set-up-the-config-file)
    * [3. Create the database](#3-create-the-database)
    * [4. Add files the patch directory](#4-add-files-the-patch-directory)
- [Developing](#developing)

## Installation

Prerequisites:
* [Go](https://golang.org)
* [Git](https://git-scm.com/)
* [PostgreSQL](https://www.postgresql.org/)

### 1. Compile the code

Assuming Go installed:

    git clone https://github.com/dcrodman/archon.git
    cd archon
    mkdir .bin
    export GOBIN=$(pwd)/.bin
    go install ./cmd/*
    
This will install the Archon server and tools to the `.bin` subdirectory in the root 
of your project's directory.

### 2. Set up the config file

If you're going to run the server out of the project's root directory (i.e. `.bin/`):

    cp setup/config.yaml .
   
Archon will also look for the config file here:

    mkdir /usr/local/etc/archon
    cp setup/config.yaml /usr/local/etc/archon

Either location will work, I'd recommend the second one so that you can also place your patch
files in this directory in order to keep everything together.

### 3. Create the database

Archon uses Postgres for persistent storage, which means you'll need to set up a Postgres database
for it to use. Once you have one ready to go: 

    createdb archondb
    psql archondb
    > CREATE USER archonadmin WITH ENCRYPTED PASSWORD 'psoadminpassword';
    > GRANT ALL ON ALL TABLES IN SCHEMA public TO archonadmin;

Feel free to choose your own credentials or database location, just make sure the settings in your 
`config.yaml` reflect them. Archon takes care of creating the tables and performing any migrations.
 
### 4. Add files the patch directory

TODO

## Developing

This is a pretty large project and pull requests, issues, and discussions are greatly appreciated!

### Suggested Development Setup
TODO

### Whirlwind tour
   TODO