#include <stdbool.h>
#include "tinycthread.h"
#include "chunk_manager.h"
#include "world_query.h"
#include "mesher.h"
#include "item.h" 
#include "matrix.h"
#include "util.h"
#include "db.h"
#include "world.h"

#define WORKERS 4
#define WORKER_IDLE 0
#define WORKER_BUSY 1
#define WORKER_DONE 2

typedef struct {
    // inputs
    int p;
    int q;
    int load;
    Map *block_maps[3][3];
    Map *light_maps[3][3];

    // outputs
    int miny;
    int maxy;
    int faces;
    GLfloat *data;
} WorkerItem;

typedef struct {
    int index;
    int state;
    thrd_t thrd;
    mtx_t mtx;
    cnd_t cnd;
    WorkerItem item;
} Worker;

struct ChunkManager {
    Chunk chunks[MAX_CHUNKS];
    Worker workers[WORKERS];
    int chunk_count;
    int create_radius;
    int render_radius;
    int delete_radius;
    int sign_radius;
};

// INTERNAL HELPERS //
static void _check_workers(ChunkManager *manager, Renderer *renderer);
static void _generate_and_upload_mesh(ChunkManager *manager, Chunk *chunk, Renderer *renderer);
static MesherOutput *_get_mesher_chunk_output(WorkerItem *item);
static void _update_chunk(Chunk *chunk, MesherOutput *mesher_output, Renderer *renderer);
static int _worker_run(void *arg);
static void _initialize_workers(ChunkManager *manager);
static void _create_chunk(ChunkManager *manager, Chunk *chunk, int p, int q);
static bool _has_lights_in_neighborhood(ChunkManager *manager, Chunk *chunk);
static void _unset_sign(ChunkManager *manager, int x, int y, int z);
static void _set_light(ChunkManager *manager, int p, int q, int x, int y, int z, int w);
static void _set_sign(
    ChunkManager *manager, int p, int q, int x, int y, int z, int face, const char *text, int dirty);
static void _unset_sign_face(ChunkManager *manager, int x, int y, int z, int face);
static void _init_chunk(ChunkManager *manager, Chunk *chunk, int p, int q);
static void _ensure_chunks_worker(ChunkManager *manager, const Camera *view, Worker *worker);
static void _map_set_func(int x, int y, int z, int w, void *arg);
static void _load_chunk(WorkerItem *item);
// ========

ChunkManager *chunk_manager_create(ChunkManagerConfig *config) {
    ChunkManager *manager = (ChunkManager *)malloc(sizeof(ChunkManager));
    if (!manager) {
        return NULL;
    }
    manager->chunk_count = 0;
    manager->create_radius = config->create_radius;
    manager->render_radius = config->render_radius;
    manager->delete_radius = config->delete_radius;
    manager->sign_radius = config->sign_radius;
    _initialize_workers(manager);
    return manager;
}
void chunk_manager_reset(ChunkManager *manager) {
    memset(manager->chunks, 0, sizeof(Chunk) * MAX_CHUNKS);
    for(int i = 0; i < MAX_CHUNKS; ++i) {
        manager->chunks[i].render_id = INVALID_RENDERABLE_OBJECT_ID;
    }
    manager->chunk_count = 0;
}
void chunk_manager_destroy(ChunkManager *manager, Renderer *renderer) {
    if (manager) {
        ChunkIterator iterator = chunk_manager_iterator_begin(manager);
        while (chunk_manager_iterator_has_next(&iterator)) {
            Chunk *chunk = chunk_manager_iterator_next(&iterator);
            map_free(&chunk->map);
            map_free(&chunk->lights);
            sign_list_free(&chunk->signs);
            renderer_delete_chunk_geometry(
                renderer, chunk->render_id);
        }
        manager->chunk_count = 0;
        free(manager);
    }
}
Chunk *chunk_manager_find_chunk(ChunkManager *manager, int p, int q) {
    for (int i = 0; i < manager->chunk_count; i++) {
        Chunk *chunk = manager->chunks + i;
        if (chunk->p == p && chunk->q == q) {
            return chunk;
        }
    }
    return NULL;
}
void chunk_manager_force_chunks_around_point(ChunkManager *manager, Renderer *renderer, float x, float z) {
    int p = chunked(x);
    int q = chunked(z);
    int r = 1;
    for (int dp = -r; dp <= r; dp++) {
        for (int dq = -r; dq <= r; dq++) {
            int a = p + dp;
            int b = q + dq;
            Chunk *chunk = chunk_manager_find_chunk(manager, a, b);
            if (chunk) {
                if (chunk->dirty) {
                    _generate_and_upload_mesh(manager, chunk, renderer);
                }
            }
            else if (manager->chunk_count < MAX_CHUNKS) {
                chunk = manager->chunks + manager->chunk_count++;
                _create_chunk(manager, chunk, a, b);
                _generate_and_upload_mesh(manager, chunk, renderer);
            }
        }
    }
}

void chunk_manager_update(ChunkManager *manager, const Camera *view, Renderer *renderer) {
    _check_workers(manager, renderer);
    chunk_manager_force_chunks_around_point(manager, renderer, view->x, view->z);
    for (int i = 0; i < WORKERS; i++) {
        Worker *worker = manager->workers + i;
        mtx_lock(&worker->mtx);
        if (worker->state == WORKER_IDLE) {
            _ensure_chunks_worker(manager, view, worker);
        }
        mtx_unlock(&worker->mtx);
    }
}
void chunk_manager_set_dirty_chunk(ChunkManager *manager, Chunk *chunk) {
    chunk->dirty = 1;
    if (_has_lights_in_neighborhood(manager, chunk))
    {
        for (int dp = -1; dp <= 1; dp++)
        {
            for (int dq = -1; dq <= 1; dq++)
            {
                Chunk *other = chunk_manager_find_chunk(manager, chunk->p + dp, chunk->q + dq);
                if (other)
                {
                    other->dirty = 1;
                }
            }
        }
    }
}
void chunk_manager_delete_distant_chunks(ChunkManager *manager, Renderer *renderer, float x, float z) {
    for (int i = 0; i < manager->chunk_count; i++)
    {
        Chunk *chunk = manager->chunks + i;
        int delete = 1;
        int p = chunked(x);
        int q = chunked(z);
        if (chebyshev_distance(p, q, chunk->p, chunk->q) < manager->delete_radius)
        {
            delete = 0;
            break;
        }

        if (delete)
        {
            map_free(&chunk->map);
            map_free(&chunk->lights);
            sign_list_free(&chunk->signs);
            if (chunk->render_id != INVALID_RENDERABLE_OBJECT_ID) {
                renderer_delete_chunk_geometry(renderer, chunk->render_id);
            }
            Chunk *other = manager->chunks + (--manager->chunk_count);
            memcpy(chunk, other, sizeof(Chunk));
        }
    }
}
void _set_block(ChunkManager *manager, int p, int q, int x, int y, int z, int w, int dirty)
{
    Chunk *chunk = chunk_manager_find_chunk(manager, p, q);
    if (chunk)
    {
        Map *map = &chunk->map;
        if (map_set(map, x, y, z, w))
        {
            if (dirty)
            {
                chunk_manager_set_dirty_chunk(manager, chunk);
            }
            db_insert_block(p, q, x, y, z, w);
        }
    }
    else
    {
        db_insert_block(p, q, x, y, z, w);
    }
    if (w == 0 && chunked(x) == p && chunked(z) == q)
    {
        _unset_sign(manager, x, y, z);
        _set_light(manager, p, q, x, y, z, 0);
    }
}
void chunk_manager_set_block(ChunkManager *manager, int x, int y, int z, int w) {
    int p = chunked(x);
    int q = chunked(z);
    _set_block(manager, p, q, x, y, z, w, 1);
    for (int dx = -1; dx <= 1; dx++)
    {
        for (int dz = -1; dz <= 1; dz++)
        {
            if (dx == 0 && dz == 0)
            {
                continue;
            }
            if (dx && chunked(x + dx) == p)
            {
                continue;
            }
            if (dz && chunked(z + dz) == q)
            {
                continue;
            }
            _set_block(manager, p + dx, q + dz, x, y, z, -w, 1);
        }
    }
}
void chunk_manager_toggle_light(ChunkManager *manager, int x, int y, int z) {
    int p = chunked(x);
    int q = chunked(z);
    Chunk *chunk = chunk_manager_find_chunk(manager, p, q);
    if (chunk) {
        Map *map = &chunk->lights;
        int w = map_get(map, x, y, z) ? 0 : 15;
        map_set(map, x, y, z, w);
        db_insert_light(p, q, x, y, z, w);
        chunk_manager_set_dirty_chunk(manager, chunk);
    }
}
void chunk_manager_set_sign(ChunkManager *manager, int x, int y, int z, int face, const char *text) {
    int p = chunked(x);
    int q = chunked(z);
    _set_sign(manager, p, q, x, y, z, face, text, 1);
}
ChunkIterator chunk_manager_iterator_begin(ChunkManager *manager) {
    ChunkIterator iterator;
    iterator.manager = manager;
    iterator.current_index = -1; // to allow while loop, I did this for convenience
    return iterator;
}
bool chunk_manager_iterator_has_next(ChunkIterator *iterator) {
    return ++iterator->current_index < iterator->manager->chunk_count;
}
Chunk *chunk_manager_iterator_next(const ChunkIterator *iterator) {
    Chunk *chunk = &iterator->manager->chunks[iterator->current_index];
    return chunk;
}
// INTERNAL HELPERS IMPLEMENTATIONS //
static void _set_sign(
    ChunkManager *manager, int p, int q, int x, int y, int z, int face, const char *text, int dirty)
{
    if (strlen(text) == 0)
    {
        _unset_sign_face(manager, x, y, z, face);
        return;
    }
    Chunk *chunk = chunk_manager_find_chunk(manager, p, q);
    if (chunk)
    {
        SignList *signs = &chunk->signs;
        sign_list_add(signs, x, y, z, face, text);
        if (dirty)
        {
            chunk->dirty = 1;
        }
    }
    db_insert_sign(p, q, x, y, z, face, text);
}
static void _unset_sign_face(ChunkManager *manager, int x, int y, int z, int face) {
    int p = chunked(x);
    int q = chunked(z);
    Chunk *chunk = chunk_manager_find_chunk(manager, p, q);
    if (chunk)
    {
        SignList *signs = &chunk->signs;
        if (sign_list_remove(signs, x, y, z, face))
        {
            chunk->dirty = 1;
            db_delete_sign(x, y, z, face);
        }
    }
    else
    {
        db_delete_sign(x, y, z, face);
    }
}
static void _init_chunk(ChunkManager *manager, Chunk *chunk, int p, int q) {
    chunk->p = p;
    chunk->q = q;
    chunk->render_id = INVALID_RENDERABLE_OBJECT_ID;
    chunk_manager_set_dirty_chunk(manager, chunk);
    SignList *signs = &chunk->signs;
    sign_list_alloc(signs, 16);
    db_load_signs(signs, p, q);
    Map *block_map = &chunk->map;
    Map *light_map = &chunk->lights;
    int dx = p * CHUNK_SIZE - 1;
    int dy = 0;
    int dz = q * CHUNK_SIZE - 1;
    map_alloc(block_map, dx, dy, dz, 0x7fff);
    map_alloc(light_map, dx, dy, dz, 0xf);
}
static void _ensure_chunks_worker(ChunkManager *manager, const Camera *view, Worker *worker) {
    int p = chunked(view->x);
    int q = chunked(view->z);
    int r = manager->create_radius;
    int start = 0x0fffffff;
    int best_score = start;
    int best_a = 0;
    int best_b = 0;
    for (int dp = -r; dp <= r; dp++) {
        for (int dq = -r; dq <= r; dq++) {
            int a = p + dp;
            int b = q + dq;
            int index = (ABS(a) ^ ABS(b)) % WORKERS;
            if (index != worker->index) {
                continue;
            }
            Chunk *chunk = chunk_manager_find_chunk(manager, a, b);
            if (chunk && !chunk->dirty) {
                continue;
            }
            int distance = MAX(ABS(dp), ABS(dq));
            int invisible = !world_is_chunk_visible(view, a, b, 0, 256);
            int priority = 0;
            if (chunk) {
                priority = manager->chunks->render_id != INVALID_RENDERABLE_OBJECT_ID && chunk->dirty;
            }
            int score = (invisible << 24) | (priority << 16) | distance;
            if (score < best_score) {
                best_score = score;
                best_a = a;
                best_b = b;
            }
        }
    }
    if (best_score == start) {
        return;
    }
    int a = best_a;
    int b = best_b;
    int load = 0;
    Chunk *chunk = chunk_manager_find_chunk(manager, a, b);
    if (!chunk) {
        load = 1;
        if (manager->chunk_count < MAX_CHUNKS) {
            chunk = manager->chunks + manager->chunk_count++;
            _init_chunk(manager, chunk, a, b);
        }
        else {
            return;
        }
    }
    WorkerItem *item = &worker->item;
    item->p = chunk->p;
    item->q = chunk->q;
    item->load = load;
    for (int dp = -1; dp <= 1; dp++) {
        for (int dq = -1; dq <= 1; dq++) {
            Chunk *other = chunk;
            if (dp || dq) {
                other = chunk_manager_find_chunk(manager, chunk->p + dp, chunk->q + dq);
            }
            if (other) {
                Map *block_map = malloc(sizeof(Map));
                map_copy(block_map, &other->map);
                Map *light_map = malloc(sizeof(Map));
                map_copy(light_map, &other->lights);
                item->block_maps[dp + 1][dq + 1] = block_map;
                item->light_maps[dp + 1][dq + 1] = light_map;
            }
            else {
                item->block_maps[dp + 1][dq + 1] = 0;
                item->light_maps[dp + 1][dq + 1] = 0;
            }
        }
    }
    chunk->dirty = 0;
    worker->state = WORKER_BUSY;
    cnd_signal(&worker->cnd);
}
static void _map_set_func(int x, int y, int z, int w, void *arg) {
    Map *map = (Map *)arg;
    map_set(map, x, y, z, w);
}
static void _load_chunk(WorkerItem *item) {
    int p = item->p;
    int q = item->q;
    Map *block_map = item->block_maps[1][1];
    Map *light_map = item->light_maps[1][1];
    create_world(p, q, _map_set_func, block_map);
    db_load_blocks(block_map, p, q);
    db_load_lights(light_map, p, q);
}
static void _check_workers(ChunkManager *manager, Renderer *renderer) {
    for (int i = 0; i < WORKERS; i++) {
        Worker *worker = manager->workers + i;
        mtx_lock(&worker->mtx);
        if (worker->state == WORKER_DONE) {
            WorkerItem *item = &worker->item;
            Chunk *chunk = chunk_manager_find_chunk(manager, item->p, item->q);
            if (chunk) {
                if (item->load) {
                    Map *block_map = item->block_maps[1][1];
                    Map *light_map = item->light_maps[1][1];
                    map_free(&chunk->map);
                    map_free(&chunk->lights);
                    map_copy(&chunk->map, block_map);
                    map_copy(&chunk->lights, light_map);
                }
                MesherOutput *temp_mesh = &((MesherOutput) {
                    .data = item->data,
                    .miny = item->miny,
                    .maxy = item->maxy,
                });
                _update_chunk(chunk, temp_mesh, renderer);
                mesher_free_output_data(temp_mesh);
            }
            for (int a = 0; a < 3; a++) {
                for (int b = 0; b < 3; b++) {
                    Map *block_map = item->block_maps[a][b];
                    Map *light_map = item->light_maps[a][b];
                    if (block_map) {
                        map_free(block_map);
                        free(block_map);
                    }
                    if (light_map) {
                        map_free(light_map);
                        free(light_map);
                    }
                }
            }
            worker->state = WORKER_IDLE;
        }
        mtx_unlock(&worker->mtx);
    }
}
// generates the chunk buffer and uploads it to the GPU for the dirty chunk
static void _generate_and_upload_mesh(ChunkManager *manager, Chunk *chunk, Renderer *renderer) {
    WorkerItem _item;
    WorkerItem *item = &_item;
    item->p = chunk->p;
    item->q = chunk->q;
    for (int dp = -1; dp <= 1; dp++) {
        for (int dq = -1; dq <= 1; dq++) {
            Chunk *other = chunk;
            if (dp || dq) {
                other = chunk_manager_find_chunk(manager, chunk->p + dp, chunk->q + dq);
            }
            if (other) {
                item->block_maps[dp + 1][dq + 1] = &other->map;
                item->light_maps[dp + 1][dq + 1] = &other->lights;
            }
            else {
                item->block_maps[dp + 1][dq + 1] = 0;
                item->light_maps[dp + 1][dq + 1] = 0;
            }
        }
    }
    MesherOutput *mesher_output = _get_mesher_chunk_output(item);
    _update_chunk(chunk, mesher_output, renderer);
    mesher_free_output(&mesher_output);
    chunk->dirty = 0;
}
// will store it appropriately in the WorkerItem passed in (more useful for async path)
static MesherOutput *_get_mesher_chunk_output(WorkerItem *item) {
    item->data = NULL;
    item->faces = 0;
    item->miny = 0;
    item->maxy = 0;
    MesherInput mesher_input;
    mesher_input.p = item->p;
    mesher_input.q = item->q;
    memcpy(mesher_input.block, item->block_maps, sizeof(item->block_maps));
    memcpy(mesher_input.light, item->light_maps, sizeof(item->light_maps));
    MesherOutput *mesher_output = mesher_compute_chunk(&mesher_input);
    if (mesher_output) {
        item->data = mesher_output->data;
        item->faces = mesher_output->faces;
        item->miny = mesher_output->miny;
        item->maxy = mesher_output->maxy;
        return mesher_output;
    } else {
        item->data = NULL;
        item->faces = 0;
        item->miny = 0;
        item->maxy = 0;
        return NULL;
    }
}
static void _update_chunk(Chunk *chunk, MesherOutput *mesher_output, Renderer *renderer) {
    chunk->miny = mesher_output->miny;
    chunk->maxy = mesher_output->maxy;
    mesher_generate_sign_mesh(mesher_output, &chunk->signs);
    renderer_upload_chunk_geometry(renderer, &chunk->render_id, mesher_output); // includes signs
}
static int _worker_run(void *arg) { 
    Worker *worker = (Worker *)arg;
    int running = 1;
    while (running) {
        mtx_lock(&worker->mtx);
        while (worker->state != WORKER_BUSY) {
            cnd_wait(&worker->cnd, &worker->mtx);
        }
        mtx_unlock(&worker->mtx);
        WorkerItem *item = &worker->item;
        if (item->load) {
            _load_chunk(item);
        }
        MesherOutput *mesh_output = _get_mesher_chunk_output(item);
        if (mesh_output) {
            mesher_free_output(&mesh_output);
        }
        mtx_lock(&worker->mtx);
        worker->state = WORKER_DONE;
        mtx_unlock(&worker->mtx);
    }
    return 0;
}
static void _initialize_workers(ChunkManager *manager) {
    for (int i = 0; i < WORKERS; i++) {
        Worker *worker = manager->workers + i;
        worker->index = i;
        worker->state = WORKER_IDLE;
        mtx_init(&worker->mtx, mtx_plain);
        cnd_init(&worker->cnd);
        thrd_create(&worker->thrd, _worker_run, worker);
    }
}
static void _create_chunk(ChunkManager *manager, Chunk *chunk, int p, int q) {
    _init_chunk(manager, chunk, p, q);

    WorkerItem _item;
    WorkerItem *item = &_item;
    item->p = chunk->p;
    item->q = chunk->q;
    item->block_maps[1][1] = &chunk->map;
    item->light_maps[1][1] = &chunk->lights;
    _load_chunk(item);
}
static bool _has_lights_in_neighborhood(ChunkManager *manager, Chunk *chunk) {
    if (!SHOW_LIGHTS)
    {
        return false;
    }
    for (int dp = -1; dp <= 1; dp++)
    {
        for (int dq = -1; dq <= 1; dq++)
        {
            Chunk *other = chunk;
            if (dp || dq)
            {
                other = chunk_manager_find_chunk(manager, chunk->p + dp, chunk->q + dq);
            }
            if (!other)
            {
                continue;
            }
            Map *map = &other->lights;
            if (map->size)
            {
                return true;
            }
        }
    }
    return false;
}
static void _unset_sign(ChunkManager *manager, int x, int y, int z) {
    int p = chunked(x);
    int q = chunked(z);
    Chunk *chunk = chunk_manager_find_chunk(manager, p, q);
    if (chunk)
    {
        SignList *signs = &chunk->signs;
        if (sign_list_remove_all(signs, x, y, z))
        {
            chunk->dirty = 1;
            db_delete_signs(x, y, z);
        }
    }
    else
    {
        db_delete_signs(x, y, z);
    }
}
static void _set_light(ChunkManager *manager, int p, int q, int x, int y, int z, int w) {
    Chunk *chunk = chunk_manager_find_chunk(manager, p, q);
    if (chunk)
    {
        Map *map = &chunk->lights;
        if (map_set(map, x, y, z, w))
        {
            chunk_manager_set_dirty_chunk(manager, chunk);
            db_insert_light(p, q, x, y, z, w);
        }
    }
    else
    {
        db_insert_light(p, q, x, y, z, w);
    }
}