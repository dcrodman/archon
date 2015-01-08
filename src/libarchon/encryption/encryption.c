/* PSO Encryption Library 
 * 
 * All included source written by Fuzziqer Software except where otherwise 
 * indicated. 
 * copyright 2004 
 */

#include <stdio.h>
#include <string.h>
#include "encryption.h"

// Internal functions (don't call these) 
unsigned long CRYPT_PC_GetNextKey(CRYPT_SETUP*);
void CRYPT_PC_MixKeys(CRYPT_SETUP*);
void CRYPT_PC_CreateKeys(CRYPT_SETUP*,uint32_t);
void CRYPT_PC_CryptData(CRYPT_SETUP*,void*,unsigned long);

unsigned long CRYPT_GC_GetNextKey(CRYPT_SETUP*);
void CRYPT_GC_MixKeys(CRYPT_SETUP*);
void CRYPT_GC_CreateKeys(CRYPT_SETUP*,uint32_t);
void CRYPT_GC_CryptData(CRYPT_SETUP*,void*,unsigned long);

void CRYPT_BB_Decrypt(CRYPT_SETUP*,void*,unsigned long);
void CRYPT_BB_Encrypt(CRYPT_SETUP*,void*,unsigned long);
void CRYPT_BB_CreateKeys(CRYPT_SETUP*,void*);

extern void CRYPT_BB_DEBUG_PrintKeys(CRYPT_SETUP *cs,char *title);
extern void CRYPT_GC_DEBUG_PrintKeys(CRYPT_SETUP* cs,char* title);
extern void CRYPT_PC_DEBUG_PrintKeys(CRYPT_SETUP* cs,char* title);

int CRYPT_CreateKeys(CRYPT_SETUP* cs,void* key,unsigned char type)
{
    memset(cs, 0, sizeof(CRYPT_SETUP));
    cs->type = type;
    switch (cs->type)
    {
      case CRYPT_PC:
        CRYPT_PC_CreateKeys(cs,*(uint32_t*)key);
        break;
      case CRYPT_GAMECUBE:
        CRYPT_GC_CreateKeys(cs,*(uint32_t*)key);
        break;
      case CRYPT_BLUEBURST:
        CRYPT_BB_CreateKeys(cs,key);
        break;
      default:
        return 0;
    }
    return 1;
}

int CRYPT_CryptData(CRYPT_SETUP* cs,void* data,unsigned long size,int encrypting)
{
    switch (cs->type)
    {
      case CRYPT_PC:
        CRYPT_PC_CryptData(cs, data, size);
        break;
      case CRYPT_GAMECUBE:
        CRYPT_GC_CryptData(cs,data,size);
        break;
      case CRYPT_BLUEBURST:
        if (encrypting) CRYPT_BB_Encrypt(cs,data,size);
        else CRYPT_BB_Decrypt(cs,data,size);
        break;
      default:
        return 0;
    }
    return 1;
}

void CRYPT_DEBUG_PrintKeys(CRYPT_SETUP* cs,char* title)
{
    switch (cs->type)
    {
      case CRYPT_PC:
        CRYPT_PC_DEBUG_PrintKeys(cs,title);
        break;
      case CRYPT_GAMECUBE:
        CRYPT_GC_DEBUG_PrintKeys(cs,title);
        break;
      case CRYPT_BLUEBURST:
        CRYPT_BB_DEBUG_PrintKeys(cs,title);
        break;
    }
}

void CRYPT_PrintData(void* ds,unsigned long data_size)
{
    unsigned char* data_source = (unsigned char*)ds;
    unsigned long x,y,off;
    char buffer[17];
    buffer[16] = 0;
    off = 0;
    printf("0000 | ");
    for (x = 0; x < data_size; x++)
    {
        if (off == 16)
        {
            memcpy(buffer,&data_source[x - 16],16);
            for (y = 0; y < 16; y++) if (buffer[y] < 0x20) buffer[y] = '.';
            printf("| %s\n%04X | ",buffer,(unsigned int)x);
            off = 0;
        }
        printf("%02X ",data_source[x]);
        off++;
    }
    buffer[off] = 0;
    memcpy(buffer,&data_source[x - off],off);
    for (y = 0; y < off; y++) if (buffer[y] < 0x20) buffer[y] = '.';
    for (y = 0; y < 16 - off; y++) printf("   ");
    printf("| %s\n",buffer);
}

