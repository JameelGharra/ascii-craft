#ifndef _ipc_h_
#define _ipc_h_

#include <stdint.h>
#include <stdbool.h>
#include "ascii_renderer.h"

#define SHM_SIZE (1024  * 1024 * 4) // ~ 4 MB (I did this to be suited for 4K resolution ASCII - which will never gonna happen)

#if defined _WIN32
    #define SHM_NAME "Local\\CraftSharedMemory"
#else
    #define SHM_NAME "/craft_shared_memory"
#endif

#define IPC_CMD_BUFFER_SIZE 256 // tried to ensure fast wrapping by picking a power of 2

typedef enum {
    IPC_CMD_NONE = 0,
    // movement with duration
    IPC_CMD_FORWARD,
    IPC_CMD_BACKWARD,
    IPC_CMD_LEFT,
    IPC_CMD_RIGHT,
    // actions
    IPC_CMD_JUMP,
    IPC_CMD_FLY,
    IPC_CMD_BUILD,
    IPC_CMD_DESTROY,
    IPC_CMD_SELECT_SLOT, // 1-9 for item selection
    // looks with angle and duration
    IPC_CMD_TURN_LEFT,  // yaw -90
    IPC_CMD_TURN_RIGHT, // yaw +90
    IPC_CMD_LOOK_UP,    // pitch +15
    IPC_CMD_LOOK_DOWN,  // pitch -15
    // casual jumps
    IPC_CMD_JUMP_FORWARD,
    IPC_CMD_JUMP_BACKWARD,
    IPC_CMD_JUMP_LEFT,
    IPC_CMD_JUMP_RIGHT,
    IPC_CMD_LOOK_LEFT,  // yaw -15
    IPC_CMD_LOOK_RIGHT, // yaw +15
} IPCCommandType;

typedef struct {
    uint32_t type;
    int32_t value;
} IPCCommandEntry;

typedef struct {
    // for video sync
    // the writer increments the seq, if odd then writing in progress, if even -> data is ready, if changed while reading then reader retries
    volatile uint32_t frame_seq;
    uint32_t width;
    uint32_t height;
    uint32_t data_len; // amount of bytes data that are valid

    // for command ring buffer
    volatile uint32_t cmd_head;
    volatile uint32_t cmd_tail;
    IPCCommandEntry commands[IPC_CMD_BUFFER_SIZE];
    
    // video payload
    uint8_t data[];
} SharedMemoryLayout;

void ipc_create();
void ipc_destroy();

void ipc_write_frame(uint8_t *buffer, uint32_t len, uint32_t w, uint32_t h);
void ipc_notify_data_not_ready();
void ipc_notify_data_ready(uint32_t w, uint32_t h, uint32_t len);
void *ipc_get_data_pointer();
bool ipc_read_command(IPCCommandEntry *out_cmd);

#endif