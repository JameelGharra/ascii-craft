#ifndef _ipc_h_
#define _ipc_h_

#include <stdint.h>
#include <stdbool.h>

#define SHM_SIZE (1024  * 1024 * 4) // ~ 4 MB (I did this to be suited for 4K resolution ASCII - which will never gonna happen)
#define SHM_NAME "local\\CraftSharedMemory"

typedef struct {
    // the writer increments the seq, if odd then writing in progress, if even -> data is ready, if changed while reading then reader retries
    volatile uint32_t frame_seq;
    uint32_t width;
    uint32_t height;

    uint32_t data_len; // amount of bytes data that are valid
    uint8_t data[];
} SharedMemoryLayout;

void ipc_create();
void ipc_destroy();

void ipc_write_frame(uint8_t *buffer, uint32_t len, uint32_t w, uint32_t h);


#endif