# Docker compose for tethealla and packet analyzer

This compose file is intended to be used mostly on Windows combined with some Windows 
executables for tethealla server, which are modified for packet_analyzer to get packets from.
See [dev guide](https://github.com/dcrodman/archon/wiki/Developer's-Guide) for more info.

This compose runs MySQL server at 8306 and packet analyzer at 8082.

## Session dumps

Analyzer generated session files are stored in the volume which 
is used for packet_analyzer container in compose file.
E.g. by default it's `/tmp/sessions` - on windows you can change it to other path like `c:/tmp/sessions`

After compose/analyzer is stopped - the folder should have session files which were generated 
by the server if there were any packets received.
