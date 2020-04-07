/*
    The PRS compressor/decompressor that this file refers to was originally
    written by Fuzziqer Software. The file was distributed with the message that
    it could be used in anything/for any purpose as long as credit was given. 
    This file included as part of archon until I feel motivated enough to
    rewrite the library in Go.

    Other than minor changes (making it compile cleanly as C) this file has been
    left relatively intact from its original distribution, which was obtained on
    June 21st, 2009 from http://www.fuzziqersoftware.com/files/prsutil.zip

    Modified June 30, 2011 by Lawrence Sebald:
    Make the code work properly when compiled for a 64-bit target.
*/

#include <stdio.h>
#include <string.h>
#include "prs.h"

////////////////////////////////////////////////////////////////////////////////

typedef struct {
    u8 bitpos;
    u8* controlbyteptr;
    u8* srcptr_orig;
    u8* dstptr_orig;
    u8* srcptr;
    u8* dstptr;
} PRS_COMPRESSOR;

static void prs_put_control_bit(PRS_COMPRESSOR* pc,u8 bit)
{
    *pc->controlbyteptr = *pc->controlbyteptr >> 1;
    *pc->controlbyteptr |= ((!!bit) << 7);
    pc->bitpos++;
    if (pc->bitpos >= 8)
    {
        pc->bitpos = 0;
        pc->controlbyteptr = pc->dstptr;
        pc->dstptr++;
    }
}

static void prs_put_control_bit_nosave(PRS_COMPRESSOR* pc,u8 bit)
{
    *pc->controlbyteptr = *pc->controlbyteptr >> 1;
    *pc->controlbyteptr |= ((!!bit) << 7);
    pc->bitpos++;
}

static void prs_put_control_save(PRS_COMPRESSOR* pc)
{
    if (pc->bitpos >= 8)
    {
        pc->bitpos = 0;
        pc->controlbyteptr = pc->dstptr;
        pc->dstptr++;
    }
}

static void prs_put_static_data(PRS_COMPRESSOR* pc,u8 data)
{
    *pc->dstptr = data;
    pc->dstptr++;
}

static u8 prs_get_static_data(PRS_COMPRESSOR* pc)
{
    u8 data = *pc->srcptr;
    pc->srcptr++;
    return data;
}

////////////////////////////////////////////////////////////////////////////////

static void prs_init(PRS_COMPRESSOR* pc,void* src,void* dst)
{
    pc->bitpos = 0;
    pc->srcptr = (u8*)src;
    pc->srcptr_orig = (u8*)src;
    pc->dstptr = (u8*)dst;
    pc->dstptr_orig = (u8*)dst;
    pc->controlbyteptr = pc->dstptr;
    pc->dstptr++;
}

static void prs_finish(PRS_COMPRESSOR* pc)
{
    prs_put_control_bit(pc,0);
    prs_put_control_bit(pc,1);

    if (pc->bitpos != 0)
    {
        *pc->controlbyteptr = ((*pc->controlbyteptr << pc->bitpos) >> 8);
    }

    prs_put_static_data(pc,0);
    prs_put_static_data(pc,0);
}

static void prs_rawbyte(PRS_COMPRESSOR* pc)
{
    prs_put_control_bit_nosave(pc,1);
    prs_put_static_data(pc,prs_get_static_data(pc));
    prs_put_control_save(pc);
}

static void prs_shortcopy(PRS_COMPRESSOR* pc,int offset,u8 size)
{
    size -= 2;
    prs_put_control_bit(pc,0);
    prs_put_control_bit(pc,0);
    prs_put_control_bit(pc,(size >> 1) & 1);
    prs_put_control_bit_nosave(pc,size & 1);
    prs_put_static_data(pc,offset & 0xFF);
    prs_put_control_save(pc);
}

static void prs_longcopy(PRS_COMPRESSOR* pc,int offset,u8 size)
{
    u8 byte1,byte2;
    if (size <= 9)
    {
        prs_put_control_bit(pc,0);
        prs_put_control_bit_nosave(pc,1);
        prs_put_static_data(pc,((offset << 3) & 0xF8) | ((size - 2) & 0x07));
        prs_put_static_data(pc,(offset >> 5) & 0xFF);
        prs_put_control_save(pc);
    } else {
        prs_put_control_bit(pc,0);
        prs_put_control_bit_nosave(pc,1);
        prs_put_static_data(pc,(offset << 3) & 0xF8);
        prs_put_static_data(pc,(offset >> 5) & 0xFF);
        prs_put_static_data(pc,size - 1);
        prs_put_control_save(pc);
    }
}

static void prs_copy(PRS_COMPRESSOR* pc,int offset,u8 size)
{
    if ((offset > -0x100) && (size <= 5))
    {
        prs_shortcopy(pc,offset,size);
    } else {
        prs_longcopy(pc,offset,size);
    }
    pc->srcptr += size;
}

////////////////////////////////////////////////////////////////////////////////

u32 prs_compress(void* source,void* dest,u32 size)
{
    PRS_COMPRESSOR pc;
    int x,y,z;
    u32 xsize;
    int lsoffset,lssize;
    u8 *src = (u8 *)source, *dst = (u8 *)dest;
    prs_init(&pc,source,dest);

    for (x = 0; x < size; x++)
    {
        lsoffset = lssize = xsize = 0;
        for (y = x - 3; (y > 0) && (y > (x - 0x1FF0)) && (xsize < 255); y--)
        {
            xsize = 3;
            if (!memcmp(src + y, src + x, xsize))
            {
                do xsize++;
                while (!memcmp(src + y, src + x, xsize) &&
                       (xsize < 256) &&
                       ((y + xsize) < x) &&
                       ((x + xsize) <= size)
                );
                xsize--;
                if (xsize > lssize)
                {
                    lsoffset = -(x - y);
                    lssize = xsize;
                }
            }
        }
        if (lssize == 0)
        {
            prs_rawbyte(&pc);
        } else {
            prs_copy(&pc,lsoffset,lssize);
            x += (lssize - 1);
        }
    }
    prs_finish(&pc);
    return pc.dstptr - pc.dstptr_orig;
}

////////////////////////////////////////////////////////////////////////////////

u32 prs_decompress(void* source,void* dest) // 800F7CB0 through 800F7DE4 in mem 
{
    u32 r0,r3,r6,r9; // 6 unnamed registers 
    u32 bitpos = 9; // 4 named registers 
    u8* sourceptr = (u8*)source;
    u8* sourceptr_orig = (u8*)source;
    u8* destptr = (u8*)dest;
    u8* destptr_orig = (u8*)dest;
    u8 *ptr_reg;
    u8 currentbyte;
    int flag;
    int32_t offset;
    u32 x,t; // 2 placed variables 

    currentbyte = sourceptr[0];
    sourceptr++;
    for (;;)
    {
        bitpos--;
        if (bitpos == 0)
        {
            currentbyte = sourceptr[0];
            bitpos = 8;
            sourceptr++;
        }
        flag = currentbyte & 1;
        currentbyte = currentbyte >> 1;
        if (flag)
        {
            destptr[0] = sourceptr[0];
            sourceptr++;
            destptr++;
            continue;
        }
        bitpos--;
        if (bitpos == 0)
        {
            currentbyte = sourceptr[0];
            bitpos = 8;
            sourceptr++;
        }
        flag = currentbyte & 1;
        currentbyte = currentbyte >> 1;
        if (flag)
        {
            r3 = sourceptr[0] & 0xFF;
            offset = ((sourceptr[1] & 0xFF) << 8) | r3;
            sourceptr += 2;
            if (offset == 0) return (u32)(destptr - destptr_orig);
            r3 = r3 & 0x00000007;
            //r5 = (offset >> 3) | 0xFFFFE000;
            if (r3 == 0)
            {
                flag = 0;
                r3 = sourceptr[0] & 0xFF;
                sourceptr++;
                r3++;
            } else r3 += 2;
            //r5 += (u32)destptr;
            ptr_reg = destptr + ((int32_t)((offset >> 3) | 0xFFFFE000));
        } else {
            r3 = 0;
            for (x = 0; x < 2; x++)
            {
                bitpos--;
                if (bitpos == 0)
                {
                    currentbyte = sourceptr[0];
                    bitpos = 8;
                    sourceptr++;
                }
                flag = currentbyte & 1;
                currentbyte = currentbyte >> 1;
                offset = r3 << 1;
                r3 = offset | flag;
            }
            offset = sourceptr[0] | 0xFFFFFF00;
            r3 += 2;
            sourceptr++;
            //r5 = offset + (u32)destptr;
            ptr_reg = destptr + offset;
        }
        if (r3 == 0) continue;
        t = r3;
        for (x = 0; x < t; x++)
        {
            //destptr[0] = *(u8*)r5;
            //r5++;
            *destptr++ = *ptr_reg++;
            r3++;
            //destptr++;
        }
    }
}

u32 prs_decompress_size(void* source)
{
    u32 r0,r3,r6,r9; // 6 unnamed registers 
    u32 bitpos = 9; // 4 named registers 
    u8* sourceptr = (u8*)source;
    u8* destptr = NULL;
    u8* destptr_orig = NULL;
    u8 *ptr_reg;
    u8 currentbyte,lastbyte;
    int flag;
    int32_t offset;
    u32 x,t; // 2 placed variables 

    currentbyte = sourceptr[0];
    sourceptr++;
    for (;;)
    {
        bitpos--;
        if (bitpos == 0)
        {
            lastbyte = currentbyte = sourceptr[0];
            bitpos = 8;
            sourceptr++;
        }
        flag = currentbyte & 1;
        currentbyte = currentbyte >> 1;
        if (flag)
        {
            sourceptr++;
            destptr++;
            continue;
        }
        bitpos--;
        if (bitpos == 0)
        {
            lastbyte = currentbyte = sourceptr[0];
            bitpos = 8;
            sourceptr++;
        }
        flag = currentbyte & 1;
        currentbyte = currentbyte >> 1;
        if (flag)
        {
            r3 = sourceptr[0];
            offset = (sourceptr[1] << 8) | r3;
            sourceptr += 2;
            if (offset == 0) return (u32)(destptr - destptr_orig);
            r3 = r3 & 0x00000007;
            //r5 = (offset >> 3) | 0xFFFFE000;
            if (r3 == 0)
            {
                r3 = sourceptr[0];
                sourceptr++;
                r3++;
            } else r3 += 2;
            //r5 += (u32)destptr;
            ptr_reg = destptr + ((int32_t)((offset >> 3) | 0xFFFFE000));
        } else {
            r3 = 0;
            for (x = 0; x < 2; x++)
            {
                bitpos--;
                if (bitpos == 0)
                {
                    lastbyte = currentbyte = sourceptr[0];
                    bitpos = 8;
                    sourceptr++;
                }
                flag = currentbyte & 1;
                currentbyte = currentbyte >> 1;
                offset = r3 << 1;
                r3 = offset | flag;
            }
            offset = sourceptr[0] | 0xFFFFFF00;
            r3 += 2;
            sourceptr++;
            //r5 = offset + (u32)destptr;
            ptr_reg = destptr + offset;
        }
        if (r3 == 0) continue;
        t = r3;
        for (x = 0; x < t; x++)
        {
            //r5++;
            ptr_reg++;
            r3++;
            destptr++;
        }
    }
}
