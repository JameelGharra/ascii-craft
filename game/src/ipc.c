#include "ipc.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static SharedMemoryLayout *shm_ptr = NULL;

#ifdef _WIN32
    #include <windows.h>
    
    static HANDLE hMapFile;

    static void print_last_error(const char *msg) {
        DWORD err = GetLastError();
        LPVOID lpMsgBuf;

        FormatMessageA(
            FORMAT_MESSAGE_ALLOCATE_BUFFER |
            FORMAT_MESSAGE_FROM_SYSTEM |
            FORMAT_MESSAGE_IGNORE_INSERTS,
            NULL,
            err,
            MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT),
            (LPSTR)&lpMsgBuf,
            0, NULL );

        fprintf(stderr, "%s (%lu): %s\n", msg, err, (char*)lpMsgBuf);

        LocalFree(lpMsgBuf);
    }
    void ipc_create() {
        hMapFile = CreateFileMappingA(
            INVALID_HANDLE_VALUE,    // use paging file
            NULL,                    // default security
            PAGE_READWRITE,          // read/write access
            0,                       // maximum object size (high-order DWORD)
            SHM_SIZE,                // maximum object size (low-order DWORD)
            SHM_NAME                 // name of mapping object
        );
        if(hMapFile == NULL) {
            print_last_error("Could not create file mapping object");
            return ;
        }
        shm_ptr = (SharedMemoryLayout*) MapViewOfFile(
            hMapFile,
            FILE_MAP_ALL_ACCESS, // read/write perm
            0,
            0,
            SHM_SIZE
        );
        if(shm_ptr == NULL) {
            print_last_error("Could not map view of file");
            CloseHandle(hMapFile);
            hMapFile = NULL;
            return ;
        }
        shm_ptr->frame_seq = 0;
        shm_ptr->width = 0;
        shm_ptr->height = 0;
        shm_ptr->data_len = 0;
    }
    void ipc_destroy() {
        if(shm_ptr) {
            UnmapViewOfFile(shm_ptr);
            shm_ptr = NULL;
        }
        if(hMapFile) {
            CloseHandle(hMapFile);
            hMapFile = NULL;
        }
    }
#else
    #include <sys/mman.h>
    #include <sys/stat.h>
    #include <fcntl.h>
    #include <unistd.h>
    #include <errno.h>

    int shm_fd = -1;

    void ipc_create() {
        shm_fd = shm,_open(SHM_NAME, O_CREAT | O_RDWR, 0666);
        if (shm_fd == -1) {
            fprintf(stderr, "Could not open shared memory object '%s' (%d): %s\n",
                    SHM_NAME, errno, strerror(errno));
            return;
        }
        if (ftruncate(shm_fd SHM_SIZE) == -1) {
            fprintf(stderr, "Could not set size for shared memory '%s' (%d): %s\n",
                    SHM_NAME, errno, strerror(errno));
            close(shm_fd);
            shm_fd = -1;
            shm_unlink(SHM_NAME);
            return;
        }
        shm_ptr = mmap(0, SHM_SIZE, PROT_READ | PROT_WRITE, MAP_SHARED, shm_fd, 0);
        if (shm_ptr == MAP_FAILED) {
            fprintf(stderr, "Could not map shared memory '%s' (%d): %s\n",
                    SHM_NAME, errno, strerror(errno));
            close(shm_fd);
            shm_fd = -1;
            shm_unlink(SHM_NAME);
            shm_ptr = NULL;
            return;
        }
        shm_ptr->frame_seq = 0;
        shm_ptr->width = 0;
        shm_ptr->height = 0;
        shm_ptr->data_len = 0;
    }

    void ipc_destroy() {
        if (shm_ptr) {
            munmap(shm_ptr, SHM_SIZE);
            shm_ptr = NULL;
        }
        if (shm_fd != -1) {
            close(shm_fd);
            shm_fd = -1;
        }
        shm_unlink(SHM_NAME);
    }

#endif

void ipc_write_frame(uint8_t *buffer, uint32_t len, uint32_t w, uint32_t h) {
    if(!shm_ptr) {
        return ;
    }
    uint32_t max_payload = SHM_SIZE - sizeof(SharedMemoryLayout);
    if(len > max_payload) {
        len = max_payload;
    }
    __sync_fetch_and_add(&shm_ptr->frame_seq, 1);
    shm_ptr->width = w;
    shm_ptr->height = h;
    shm_ptr->data_len = len;
    memcpy(shm_ptr->data, buffer, len);
    __sync_fetch_and_add(&shm_ptr->frame_seq, 1); // finished writing
}
