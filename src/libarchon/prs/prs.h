/*
    The PRS compressor/decompressor that this file refers to was originally
    written by Fuzziqer Software. The file was distributed with the message that
    it could be used in anything/for any purpose as long as credit was given. This
    version of the library was modified by Lawrence of Sylverant to support 64 bit
    architectures and included as part of archon until I feel motivated enough to
    rewrite the library in Go.
*/

#include <inttypes.h>

typedef uint32_t u32;
typedef uint8_t u8;

/* compresses data using the PRS scheme.
 * This function is not based on Sega's compression routine; it was written
 * by Fuzziqer Software. It's not as efficient as Sega's, but it compresses
 * rather well.
 *
 * Arguments:
 *     void* source - data to be compressed
 *     void* dest - buffer for compressed data
 *     unsigned long size - size of the uncompressed data
 *
 * Return value: size of the compressed data
 *
 * Notes:
 *     There's no way to tell exactly how large the compressed data will be;
 *     it is recommended that the destination buffer be 9/8 the size of the
 *     source buffer (yes, larger), although it is highly unlikely that the
 *     compressed data will be larger than the uncompressed data.
 */
u32 prs_compress(void *source, void *dest, u32 size);

/* decompresses data that was compressed using the PRS scheme.
 * This function was reverse-engineered from Sega's Phantasy Star Online
 * Episode III.
 *
 * Arguments:
 *     void* source - data to be decompressed
 *     void* dest - buffer for decompressed data
 *
 * Return value: size of the decompressed data
 *
 * Notes:
 *     Do not call this function without making sure the destination buffer can
 *     hold the required amount of data. Use the following function to check
 *     the size of the decompressed data. 
 */
u32 prs_decompress(void *source, void *dest);

/* checks the original size of data that was compressed using PRS.
 * This function was reverse-engineered from Sega's Phantasy Star Online
 * Episode III.
 *
 * Arguments:
 *     void* source - data to check
 *
 * Return value: size of the decompressed data
 */
u32 prs_decompress_size(void *source);
