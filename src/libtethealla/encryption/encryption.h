/* PSO Encryption Library
 * 
 * All included source written by Fuzziqer Software except where otherwise
 * indicated.
 * copyright 2004
 */

#ifndef ENCRYPTION_H
#define ENCRYPTION_H

#include <inttypes.h>

// Supported encryption types 
#define CRYPT_GAMECUBE   0 // 521-key encryption used in PSOGC and PSOX 
#define CRYPT_BLUEBURST  1 // 1042-key encryption used in PSOBB 
#define CRYPT_PC         2 // 56-key encryption used in PSODC and PSOPC 

// Encryption data struct 
typedef struct {
    uint32_t type; // what kind of encryption is this? 
    uint32_t keys[1042]; // encryption stream 
    uint32_t pc_posn; // PSOPC crypt position 
    uint32_t* gc_block_ptr; // PSOGC crypt position 
    uint32_t* gc_block_end_ptr; // PSOGC key end pointer 
    uint32_t gc_seed; // PSOGC seed used 
    uint32_t bb_posn; // BB position (not used) 
    uint32_t bb_seed[12]; // BB seed used 
} CRYPT_SETUP;
 
/* int CRYPT_CreateKeys(CRYPT_SETUP* cs,void* key,unsigned char type)
 * 
 *   Initalizes a CRYPT_SETUP to be used to encrypt and decrypt data. 
 * 
 *   Arguments: 
 * 
 *     CRYPT_SETUP* cs 
 *         Pointer to the CRYPT_SETUP structure to prepare. 
 * 
 *     void* key 
 *         Pointer to the encryption key. For CRYPT_PC and CRYPT_GAMECUBE, this
 *         should point to a single 32-bit value (an unsigned long). For 
 *         CRYPT_BLUEBURST, this should point to a 48-byte array to be used as
 *         the key. 
 * 
 *     unsigned char type 
 *         Defines the type of encryption to use. Valid types: CRYPT_GAMECUBE, 
 *         CRYPT_BLUEBURST, and CRYPT_GAMECUBE. 
 * 
 *   Return value:
 *     The function returns 1 if the operation succeeded, or 0 if an
 *     invalid encryption type was given. 
 */
extern int CRYPT_CreateKeys(CRYPT_SETUP* cs, void* key, unsigned char type);

/* int CRYPT_CryptData(CRYPT_SETUP* cs,void* data,unsigned long size,
 *                     int encrypting)
 * 
 *   Encrypts or decrypts data. 
 * 
 *   Arguments: 
 * 
 *     CRYPT_SETUP* cs 
 *         Pointer to the CRYPT_SETUP structure to use. 
 * 
 *     void* data 
 *         Pointer to the data to be processed. 
 * 
 *     unsigned long size 
 *         Size of the data to be processed. 
 * 
 *     int encrypting 
 *         1 if the data is to be encrypted, 0 if it is to be decrypted. 
 *         Ignored unless the type of the given CRYPT_SETUP is CRYPT_BLUEBURST. 
 * 
 *   Return value:
 *     The function returns 1 if the operation succeeded, or 0 if an
 *     invalid encryption type was given. 
 */
extern int CRYPT_CryptData(CRYPT_SETUP* cs, void* data, unsigned long size,
                    int encrypting);


/* void CRYPT_PrintData(void* ds,unsigned long data_size)
 * 
 *   Prints a segment of raw data to the console usinf printf, both as
 *   hexadecimal and ASCII. 
 * 
 *   Arguments: 
 * 
 *     void* ds 
 *         Pointer to the data to be printed. 
 * 
 *     unsigned long data_size 
 *         Size of the data to be printed. 
 * 
 *   Return value: none 
 */
extern void CRYPT_PrintData(void* ds,unsigned long data_size);

#endif /* !ENCRYPTION_H */
