# How the Game Works

The PSOBB client expects there to be four logically distinct servers with which it can communicate: the Login server, the Character server, the Ship server, and the Block server. These servers are disambiguated by port and as the user makes selections the client connects to each of these servers in sequence. 

## Patch Server
Most server implementations break this up into two distinct parts; Archon does the same. 
### Patch
This basically does nothing except act as the first point of contact and provide the IP address of the Data server.

Example:
```
server -> client
Type: 0002
Size: 004C (76) bytes
client -> server
Type: 0002
Size: 0004 (4) bytes
server -> client
Type: 0004
Size: 0004 (4) bytes
client -> server
Type: 0004
Size: 0070 (112) bytes
server -> client
Type: 0013
Size: 0020 (32) bytes
server -> client
Type: 0014
Size: 000C (12) bytes
```

### Data
Optionally authenticates the client (credentials are provided but you're free not to check them) and works with the client to check the consistency of all of the files. Basically it sends a bunch of checksums and it's up to the server to validate them. This server also sends the client any files that need to be updated (i.e. don't match the checksum or don't exist).

Example (no files to update):
```
server -> client
Type: 0002
Size: 004C (76) bytes
client -> server
Type: 0002
Size: 0004 (4) bytes
server -> client
Type: 0004
Size: 0004 (4) bytes
client -> server
Type: 0004
Size: 0070 (112) bytes
server -> client
Type: 000B
Size: 0004 (4) bytes
server -> client
Type: 0009
Size: 0044 (68) bytes
server -> client
Type: 000C
Size: 0028 (40) bytes
server -> client
Type: 000D
Size: 0004 (4) bytes
client -> server
Type: 000F
Size: 0010 (16) bytes
client -> server
Type: 0010
Size: 0004 (4) bytes
server -> client
Type: 0012
Size: 0004 (4) bytes
```

### Login Server
Authenticates the client. The client sends the login credentials and the server checks them against the database, sending any failures back to the client as one of a bunch of error codes that translate to messages that appear in the dialog window. It also sets an arbitrary string of bytes that can be echo'd back to the character server as a "signature" to prove the client was authenticated.

Example (successful auth):
```
server -> client
Type: 0003
Size: 00C8 (200) bytes
client -> server
Type: 0093
Size: 00B4 (180) bytes
server -> client
Type: 00E6
Size: 0048 (72) bytes
server -> client
Type: 0019
Size: 0010 (16) bytes
```

## Character Server
This one is beefy and consists of three distinct "phases" - the client will disconnect and reconnect three times depending on the actions the user takes.

#### Phase 1: Options, Guildcard Data, Parameters, Character previews
Basically just a gigantic data dump of a bunch of files and the Guildcard data (friend list, blocked players, etc.). This is where the contents of the `parameters` directory in a server deployment are send to the client. Also where the "previews" for every character slot are sent so that the client can display the character metadata.

Example:
```
server -> client
Type: 0003
Size: 00C8 (200) bytes
server -> client
Type: 0003
Size: 00C8 (200) bytes
client -> server
Type: 0093
Size: 00B4 (180) bytes
server -> client
Type: 00E6
Size: 0044 (68) bytes
client -> server
Type: 00E0
Size: 0008 (8) bytes
server -> client
Type: 00E2
Size: 0AF8 (2808) bytes
client -> server
Type: 00E3
Size: 0010 (16) bytes
server -> client
Type: 00E5
Size: 0088 (136) bytes
client -> server
Type: 00E3
Size: 0010 (16) bytes
server -> client
Type: 00E4
Size: 0010 (16) bytes
client -> server
Type: 00E3
Size: 0010 (16) bytes
server -> client
Type: 00E4
Size: 0010 (16) bytes
client -> server
Type: 00E3
Size: 0010 (16) bytes
server -> client
Type: 00E4
Size: 0010 (16) bytes
client -> server
Type: 01E8
Size: 0010 (16) bytes
server -> client
Type: 02E8
Size: 000C (12) bytes
client -> server
Type: 03E8
Size: 0008 (8) bytes
server -> client
Type: 01DC
Size: 0014 (20) bytes
client -> server
Type: 03DC
Size: 0014 (20) bytes
server -> client
Type: 02DC
Size: 6810 (26640) bytes
client -> server
Type: 03DC
Size: 0014 (20) bytes
server -> client
Type: 02DC
Size: 6810 (26640) bytes
client -> server
Type: 03DC
Size: 0014 (20) bytes
server -> client
Type: 02DC
Size: 05A0 (1440) bytes
client -> server
Type: 03DC
Size: 0014 (20) bytes
client -> server
Type: 04EB
Size: 0008 (8) bytes
server -> client
Type: 01EB
Size: 02B4 (692) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 680C (26636) bytes
client -> server
Type: 03EB
Size: 0008 (8) bytes
server -> client
Type: 02EB
Size: 3239 (12857) bytes
client -> server
Type: 0005
Size: 0008 (8) bytes
```

#### Phase 2: Character selection
The user can pick one of the characters and advance to ship selection, or they can select "Dressing Room" or "Recreate". Both of these latter options will trigger the next phase, otherwise selecting an existing character will advance to ship selection.

Example:
```
server -> client
Type: 0003
Size: 00C8 (200) bytes
client -> server
Type: 0093
Size: 00B4 (180) bytes
server -> client
Type: 00E6
Size: 0044 (68) bytes
client -> server
Type: 00E3
Size: 0010 (16) bytes
server -> client
Type: 00E4
Size: 0010 (16) bytes
client -> server
Type: 0005
Size: 0008 (8) bytes
```

#### Phase 3: (Optional) Dressing Room / Recreate
User has the option of changing some properties of their character or creating a new character in that slot, replacing the past one entirely.

Example:
```
TODO
```

#### Phase 4: Ship selection
Sends the client the list of ships from which they can select, which contains the IP address and port of the ship server. Selecting one of these will cause the client to disconnect and connect to the ship server.

Example:
```
server -> client
Type: 0003
Size: 00C8 (200) bytes
client -> server
Type: 0093
Size: 00B4 (180) bytes
server -> client
Type: 00E6
Size: 0044 (68) bytes
server -> client
Type: 00B1
Size: 0024 (36) bytes
server -> client
Type: 00A0
Size: 0068 (104) bytes
server -> client
Type: 00EE
Size: 0050 (80) bytes
client -> server
Type: 0010
Size: 0010 (16) bytes
server -> client
Type: 0019
Size: 0010 (16) bytes
```

## Ship Server
TODO

Example:
```
```

## Block Server
TODO


Example:
```
```
