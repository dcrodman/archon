## What do you need help with?
My overall strategy for the server has evolved over time but the general approach is to try to get as many packets at least loosely implemented as possible. To that end, I spend my time working on one of three areas.

### Implementing new packets
This is what I spend the majority of my time working on, at least until Archon can do the very basic game flows. I mostly use Tethealla as a reference since it's the most feature complete Blue Burst server. The cycle looks something like this:
  1. Run a PSOBB client against Archon
  2. Client halts, crashes, or generally does something unexpected
  3. Figure out what packet Archon is missing by comparing to logs from Tethealla
  5. Implement whatever is required to properly send that packet (either by making my own guesses or by reading Tethealla/Sylverant/Newserv to figure out what data is in there)
  6. Commit the code to Archon after testing all of it
  7. Repeat

### Improving the handling of existing packets
Given the goal of just building in the basic support for as many packets as possible, I often skim over things or leave chunks of packets unused. Sometimes this is because the other servers do it, other times because I'm just punting it for later. This is probably the biggest help for others to come in and fill in since it's easier to carve out a small chunk and dive deep into what the other servers do.

### Improving development tools
There are a few tools I've built to try to make working on the server easier. The most useful one by far is the packer analyzer, which can record packets exchanged between the client and server and spit out the packets in useful formats. Every now and then I'll add features to add more information or generally try to make it easier to figure out what packets do.

## Setting up
Your setup doesn't have to look like mine but for reference my personal setup consists of:
  * MacOS
  * Goland/VSCode
  * A local Postgres instance for archon
  * Supporting files (config, patches, parameters, etc. in `/usr/local/etc/archon`) 
  * A Windows VM running in Parallels with
    + Visual Studio Community (tethealla was built with VC++ and it's impossible to open VC++ projects with anything else)
    + A custom tethealla codebase with packet logging, comments, light refactoring, etc.
    + Sylverant codebase, mostly for reference (but I did have this working in a previous VM)
    + Newserv codebase, also for reference
    + A PSOBB client patched to point to my Macbook's subnet IP (to connect to Archon)
    + A PSOBB client patched to point to localhost (to connect to Tethealla)
  * More often than not, an instance of the packet analyzer tool running (see the ([Tools](#tools)) section)

A note about the `/usr/local/etc/archon` - as I develop it's obviously easiest to be able to run the `cmd/server` directly from an IDE for the sake of attaching a debugger and not needing intermediate steps. To support this I keep everything that would be needed for a production Archon instance in that directory outside the codebase so that I don't have to tamper with the checked in files and they can remain there as defaults. Whenever I add new config values I always add to `setup/config.yaml` before altering my own.

## Tools
### cmd/analyzer
I wish I'd thought to build this tool back in 2014, it's by far the most useful one I've ever written for this project. Essentially this utility exist acts as a mechanism for capturing packet logs in a structured format so that additional tools can be written to spot differences in what Archon is doing vs the other servers.

Quick summary of the modes in `man`ish form:
```
packet_analyzer [mode] [mode args]
    capture (default):
        Starts an HTTP server as well as a TCP server that can accept packet data and convert it
        into a common format. Archon already knows how to send to this (just set debug_mode: true 
        and an address for packet_analyzer_address in config.yaml) but the TCP server is a little
        more rudimentary for the sake of communicating with it from C. See code below.

        This server collects all packet logs over the course of one or more server runs and generates
        *.session files containing formatted JSON data with the packet data and some metadata. This is
        done automatically on exit, so when you're done just Ctrl-C the server.

    summarize [session file...]:
        Takes one or more .session files as arguments and generates a shortened version of each packet.
        Generates a session_file]_summary.txt file for each session file input.

    compact [session file...]:
        Takes one or more .session files as arguments and generates a more human readable form of each
        packet sent alongside any printable characters. Generates a [session_file]_comact.txt file for
        each session file input.

    aggregate [session file...]:
        Takes one or more .session files as arguments, aggregates them in a fixed order, and renders a
        Markdown file (with collapsible sections if you pass `-collapse`) with all of the packets.
```
The best about about this tool is being able to combine the summary/compact functions with tools like `diff` to show exactly what Archon sent vs what the other servers sent.

### cmd/patcher
Handy utility for patching PSOBB executables. I used this for generating the clients in my setup above. Usage instructions are in both the package docs as well as README.md.

## Tethealla
I have a fairly heavily modified/annotated version of Tethealla that I use for reference. You can either ask for a copy of mine, or use the original version and add any other modifications/annotations yourself. 

In order to get Tethealla to send packets to the analyzer for instance, you need to add the following code to the encryptcopy, decryptcopy, and welcome messages:
```c
typedef struct st_packet_analyzer_pkt {
	char server_name[10];
	char session_name[10];
	char source[10];
	char destination[10];
} packet_analyzer_pkt;

void send_packet_analyzer_packet(
	const unsigned char* pkt,
	int size,
	const char* session_id,
	const char* src,
	const char* dest
) {
	packet_analyzer_pkt p;
	memcpy(p.server_name, "tethealla", 9);
	memcpy(p.session_name, session_id, strlen(session_id));
	memcpy(p.source, src, strlen(src));
	memcpy(p.destination, dest, strlen(dest));

	int port = 7001; // port on which the packet_analyzer TCP handler is listening
	char* host = ""; // your IP address here
	int sockfd; 
	struct sockaddr_in serv_addr;

	if ((sockfd = socket(AF_INET, SOCK_STREAM, 0)) < 0) {
		return;
	}

	serv_addr.sin_family = AF_INET;
	serv_addr.sin_port = htons(port);
	serv_addr.sin_addr.s_addr = inet_addr(host);

	if ((connect(sockfd, &serv_addr, sizeof(serv_addr)) < 0)) {
		return;
	}

	int totalSize = sizeof(struct st_packet_analyzer_pkt) + size;

	unsigned char* buffer = (unsigned char*)calloc(totalSize, sizeof(unsigned char));
	memcpy(buffer, (const unsigned char*)&p, sizeof(p));
	memcpy(buffer + sizeof(p), pkt, size);

	int bytes, sent = 0; 
	while (sent < totalSize) {
		bytes = send(sockfd, buffer + sent, totalSize - sent, 0);
		if (bytes < 0) {
			break;
		}
		sent += bytes;
	}

	closesocket(sockfd);
	free(buffer);
}
```
