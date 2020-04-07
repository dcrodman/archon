# Archon

Private server implementation for Sega's Phantasy Star Online Blue Burst. 
Credit is owed to the authors of [Tethealla](http://pioneer2.net), 
[Sylverant](http://sylverant.net), and [Newserv](http://www.fuzziqersoftware.com), 
whose servers I'm studying as I write Archon.

The goal of this project is to build a configurable, high-performing, and scalable
PSOBB server that can be run across multiple platforms with little setup overhead. 
The project is currently in development and changing rapidly while I piece together 
the PSO protocol and develop a core architecture.

Forks, bug fixes, issue reports, explanations of some of the client's bizarre behavior, 
etc. are more than welcome! I try to keep the development pretty open but if you have
questions feel free to open an issue.

## Installation

### 1. Compile the code

With Go 1.12 or later installed:

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

Feel free to choose your own credentials or database location, just make sure the 
settings in your `config.yaml` reflect them. 

    createdb archondb
    psql archondb
    > CREATE USER archonadmin WITH ENCRYPTED PASSWORD 'psoadminpassword';
    > GRANT ALL ON ALL TABLES IN SCHEMA public TO archonadmin;

Archon takes care of creating the tables and performing any migrations.
 
### 4. Add files the patch directory

TODO