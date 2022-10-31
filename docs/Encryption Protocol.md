# Encryption Protocol

(A copy and paste of my jumbled notes)

In a nutshell, PSOBB's encryption is a slightly customized Blowfish cipher operating in ECB mode.

Blowfish terminology 
* P entry: 4-byte 18 entry sub-key array
* S box: four 4-byte 256 entry arrays
* Key: max length of 56 bytes

Fuzziqer's library 
* First 72 bytes (18 uint32's) of bbtable = P entry
* Fuzzier hardcodes this
* Tethealla does some re-calculation to reverse the keys. The last four bytes are inverted and then placed n the most significant 16 bits, then the second half is inverted and placed in the least significant 16 bits
* Remaining 4096 bytes (1024 uint32's) = combined S boxes (key schedule)
* vectors = key, with a custom fixed salt

initial key schedule (bbtable) = 4096 bytes   
subkey array (size 18) = 72 bytes    
vectors generated for each cipher = 48 bytes (384 bits)   

Key Schedule Initialization  
* Each byte of the key (0...48, intervals of 3) is XOR'd by three fixed values for a salt
* Different from standard blowfish
* The P tables match in the source and the PSOBB client, but the client does some scrambling operation to de-obfuscate them. For each 4-byte entry, the lower 2 bytes are reversed and shifted to the most significant 16 bits. The original upper 16 bits are then XOR'd with the reversed bytes and stuck in the lower 16 bits. This new P table becomes the client's P table for use with the cipher. 
	•	Different from standard blowfish
* Updated P table is XOR'd with the previously salted key by incrementally combining adjacent keys (shifted into position) for a total of four bytes and wrapping around at the key length
   - ex. (essentially) p[0] ^=  key[0] | key[1] | key[2] | key[3] (positions 0xAABBCCDD); later..                         p[1] ^= key[47] | key[48] | key[0] | key[1]

## Client Encryption Keys

Blowfish parameters divided into 2 separate (LE) chunks. To locate in an exe, just look for the LE occurrence of the start and end of the P table (which is one entry) and the start and end of the S tables (which are the second entry)

Offsets for version TethVer 1.25.10: 
* 0x572E00 (72 bytes - the P entry)
* 0x571D00 (4096 bytes - the combined S boxes)

Offsets for version TethVer 1.25.13
* 0x574c2a (72 bytes - P entry)
* 0x573b2a (4096 bytes - the combined S boxes)
