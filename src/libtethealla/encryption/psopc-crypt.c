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

void CRYPT_PC_MixKeys(CRYPT_SETUP* pc)
{
    uint32_t esi,edi,eax,ebp,edx;
    edi = 1;
    edx = 0x18;
    eax = edi;
    while (edx > 0)
    {
        esi = pc->keys[eax + 0x1F];
        ebp = pc->keys[eax];
        ebp = ebp - esi;
        pc->keys[eax] = ebp;
        eax++;
        edx--;
    }
    edi = 0x19;
    edx = 0x1F;
    eax = edi;
    while (edx > 0)
    {
        esi = pc->keys[eax - 0x18];
        ebp = pc->keys[eax];
        ebp = ebp - esi;
        pc->keys[eax] = ebp;
        eax++;
        edx--;
    }
}

void CRYPT_PC_CreateKeys(CRYPT_SETUP* pc, uint32_t val)
{
    uint32_t esi,ebx,edi,eax,edx,var1;
    esi = 1;
    ebx = val;
    edi = 0x15;
    pc->keys[56] = ebx;
    pc->keys[55] = ebx;
    while (edi <= 0x46E)
    {
        eax = edi;
        var1 = eax / 55;
        edx = eax - (var1 * 55);
        ebx = ebx - esi;
        edi = edi + 0x15;
        pc->keys[edx] = esi;
        esi = ebx;
        ebx = pc->keys[edx];
    }
    CRYPT_PC_MixKeys(pc);
    CRYPT_PC_MixKeys(pc);
    CRYPT_PC_MixKeys(pc);
    CRYPT_PC_MixKeys(pc);
    pc->pc_posn = 56;
}

static uint32_t CRYPT_PC_GetNextKey(CRYPT_SETUP* pc)
{    
    uint32_t re;
    if (pc->pc_posn == 56)
    {
        CRYPT_PC_MixKeys(pc);
        pc->pc_posn = 1;
    }
    re = pc->keys[pc->pc_posn];
    pc->pc_posn++;
    return re;
}

void CRYPT_PC_CryptData(CRYPT_SETUP* pc,void* data,unsigned long size)
{
    uint32_t x, tmp;
    for (x = 0; x < size; x += 4) {
        tmp = *((uint32_t *)(data + x));
        tmp = LE32(tmp) ^ CRYPT_PC_GetNextKey(pc);
        *((uint32_t *)(data + x)) = LE32(tmp);
    }
}

void CRYPT_PC_DEBUG_PrintKeys(CRYPT_SETUP* cs,char* title)
{
    unsigned long x,y;
    printf("\n%s\n### ###+0000 ###+0001 ###+0002 ###+0003 ###+0004 ###+0005 ###+0006 ###+0007\n",title);
    for (x = 0; x < 7; x++)
    {
        printf("%03lu",x * 8);
        for (y = 0; y < 8; y++) printf(" %08X",cs->keys[(x * 8) + y]);
        printf("\n");
    }
}

