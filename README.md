Archon
===========

Private server implementation for Sega's Phantasy Star Online Blue Burst. 
Credit is owed to the authors of [Tethealla](http://pioneer2.net), 
[Newserv](http://www.fuzziqersoftware.com), and [Sylverant](http://sylverant.net), 
whose servers I'm studying as I write Archon.

The goal of this project is to build a configurable, high-performing, and scalable
PSOBB server that can be run across multiple platforms with little setup overhead. 
The project is currently in its early stages and changing rapidly while I piece 
together the PSO protocol and develop a core archiecture.

Forks, bug fixes, issue reports, etc. are welcome!

Installation
===========

Detailed instructions can be found [on the wiki](https://github.com/dcrodman/archon/wiki/Installation).

The project is built using the standard Go language toolchain, which you must 
install in order to compile and run the project. For installation instructions, 
visit the [Golang website](http://golang.org/).

With Go installed and your GOPATH set, you can fetch the project:

    go get github.com/dcrodman/archon

Server setups with one ship can just compile and run a single binary:

    go install github.com/dcrodman/archon
    $GOPATH/bin/archon