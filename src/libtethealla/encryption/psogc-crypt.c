// Reversed from the original PPC ASM by Fuzziqer Software, 2004 
// Modified by Lawrence Sebald (2009) to run properly when compiled for a 64-bit
// machine and (2011) to run properly on a big endian machine.

#include <stdio.h>
#include "encryption.h"

#if defined(__BIG_ENDIAN__) || defined(WORDS_BIGENDIAN)
#define LE32(x) (((x >> 24) & 0x00FF) | \
                 ((x >>  8) & 0xFF00) | \
                 ((x & 0xFF00) <<  8) | \
                 ((x & 0x00FF) << 24))
#else
#define LE32(x) x
#endif

////////////////////////////////////////////////////////////////////////////////
// GameCube Encryption Source 

void CRYPT_GC_MixKeys(CRYPT_SETUP* cs)
{
    uint32_t r0,r4,*r5,*r6,*r7;

    cs->gc_block_ptr = cs->keys;
    r5 = cs->keys;
    r6 = &(cs->keys[489]);
    r7 = cs->keys;

    while (r6 != cs->gc_block_end_ptr)
    {
        r0 = *(uint32_t*)r6;
        r6++;
        r4 = *(uint32_t*)r5;
        r0 ^= r4;
        *(uint32_t*)r5 = r0;
        r5++;
    }

    while (r5 != cs->gc_block_end_ptr)
    {
        r0 = *(uint32_t*)r7;
        r7++;
        r4 = *(uint32_t*)r5;
        r0 ^= r4;
        *(uint32_t*)r5 = r0;
        r5++;
    }
}

unsigned long CRYPT_GC_GetNextKey(CRYPT_SETUP* cs)
{
    cs->gc_block_ptr++;
    if (cs->gc_block_ptr == cs->gc_block_end_ptr) CRYPT_GC_MixKeys(cs);
    return *cs->gc_block_ptr;
}

void CRYPT_GC_CreateKeys(CRYPT_SETUP* cs,uint32_t seed)
{
    uint32_t x,y,basekey,*source1,*source2,*source3;
    basekey = 0;

    cs->gc_seed = seed;

    cs->gc_block_end_ptr = &(cs->keys[521]);
    cs->gc_block_ptr = (uint32_t*)cs->keys;

    for  (x = 0; x <= 16; x++)
    {
        for (y = 0; y < 32; y++)
        {
            seed = seed * 0x5D588B65;
            basekey = basekey >> 1;
            seed++;
            if (seed & 0x80000000) basekey = basekey | 0x80000000;
            else basekey = basekey & 0x7FFFFFFF;
        }
        *cs->gc_block_ptr = basekey;
        cs->gc_block_ptr = (uint32_t*)((uint8_t*)cs->gc_block_ptr + 4);
    }
    source1 = &(cs->keys[0]);
    source2 = &(cs->keys[1]);
    cs->gc_block_ptr = (uint32_t*)((uint8_t*)cs->gc_block_ptr - 4);
    (*cs->gc_block_ptr) = (((cs->keys[0] >> 9) ^ (*cs->gc_block_ptr << 23)) ^ cs->keys[15]);//cs->keys[15]);
    source3 = cs->gc_block_ptr;
    cs->gc_block_ptr = (uint32_t*)((uint8_t*)cs->gc_block_ptr + 4);
    while (cs->gc_block_ptr != cs->gc_block_end_ptr)
    {
        *cs->gc_block_ptr = (*source3 ^ (((*source1 << 23) & 0xFF800000) ^ ((*source2 >> 9) & 0x007FFFFF)));
        cs->gc_block_ptr = (uint32_t*)((uint8_t*)(cs->gc_block_ptr) + 4);
        source1 = (uint32_t*)((uint8_t*)source1 + 4);
        source2 = (uint32_t*)((uint8_t*)source2 + 4);
        source3 = (uint32_t*)((uint8_t*)source3 + 4);
    }
    CRYPT_GC_MixKeys(cs);
    CRYPT_GC_MixKeys(cs);
    CRYPT_GC_MixKeys(cs);
    cs->gc_block_ptr = &(cs->keys[520]);
}

void CRYPT_GC_CryptData(CRYPT_SETUP* c,void* data,unsigned long size)
{
    uint32_t *address_start,*address_end, tmp;

    address_start = (uint32_t*)data;
    address_end = (uint32_t*)((uint8_t*)data + size);

    while (address_start < address_end)
    {
        tmp = CRYPT_GC_GetNextKey(c);
        *address_start = *address_start ^ LE32(tmp);
        address_start++;
    }
}

void CRYPT_GC_DEBUG_PrintKeys(CRYPT_SETUP* cs,char* title)
{
    uint32_t x,y;
    printf("\n%s\n### ###+0000 ###+0001 ###+0002 ###+0003 ###+0004 ###+0005 ###+0006 ###+0007\n",title);
    for (x = 0; x < 66; x++)
    {
        printf("%03u",x * 8);
        for (y = 0; y < 8; y++) printf(" %08X",cs->keys[(x * 8) + y]);
        printf("\n");
    }
}

