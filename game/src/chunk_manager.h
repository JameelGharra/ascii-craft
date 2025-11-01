#ifndef _chunk_manager_h_
#define _chunk_manager_h_

#include <stdbool.h>
#include "config.h"
#include "player.h"
#include "renderer.h"
#include "chunk.h"
#include "camera.h"

typedef struct {
    int create_radius;
    int render_radius;
    int delete_radius;
    int sign_radius;
} ChunkManagerConfig;

typedef struct ChunkManager ChunkManager;

typedef struct {
    ChunkManager *manager;
    int current_index;
} ChunkIterator;


ChunkManager *chunk_manager_create(ChunkManagerConfig *config);
void chunk_manager_reset(ChunkManager *manager);
void chunk_manager_destroy(ChunkManager *manager, Renderer *renderer);

Chunk *chunk_manager_find_chunk(ChunkManager *manager, int p, int q);
void chunk_manager_force_chunks_around_point(ChunkManager *manager, Renderer *renderer, float x, float z);
void chunk_manager_update(ChunkManager *manager, const Camera *view, Renderer *renderer);
void chunk_manager_set_dirty_chunk(ChunkManager *manager, Chunk *chunk);
void chunk_manager_delete_distant_chunks(ChunkManager *manager, Renderer *renderer, float x, float z);
void chunk_manager_set_block(ChunkManager *manager, int x, int y, int z, int w);
void chunk_manager_toggle_light(ChunkManager *manager, int x, int y, int z);
void chunk_manager_set_sign(ChunkManager *manager, int x, int y, int z, int face, const char *text);

ChunkIterator chunk_manager_iterator_begin(ChunkManager *manager);
bool chunk_manager_iterator_has_next(ChunkIterator *iterator);
Chunk *chunk_manager_iterator_next(const ChunkIterator *iterator);

#endif