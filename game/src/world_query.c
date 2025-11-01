#include <stdlib.h>
#include <math.h>
#include "world_query.h"
#include "item.h"
#include "util.h"
#include "chunk_manager.h"

// INTERNAL HELPERS //
static int collide(const WorldQuery *world, int height, float *x, float *y, float *z);
static void get_sight_vector(float rx, float ry, float *vx, float *vy, float *vz);
static int _hit_test(
    Map *map, float max_distance, int previous, float x, float y, float z,
    float vx, float vy, float vz,
    int *hx, int *hy, int *hz);
// ========


WorldQuery *world_query_create(ChunkManager *world_context) {
    if (!world_context) {
        return NULL;
    }
    WorldQuery *world_query = (WorldQuery*)malloc(sizeof(WorldQuery));
    if (!world_query) {
        return NULL;
    }
    world_query->world_context = world_context;
    return world_query;
}

void world_query_free(WorldQuery *world_query) {
    if (world_query) {
        free(world_query);

    }
}

int world_get_highest_block(const WorldQuery *world, float x, float z) {
    int result = -1;
    int nx = roundf(x);
    int nz = roundf(z);
    int p = chunked(x);
    int q = chunked(z);
    Chunk *chunk = chunk_manager_find_chunk(world->world_context, p, q);
    if (chunk)
    {
        Map *map = &chunk->map;
        MAP_FOR_EACH(map, ex, ey, ez, ew)
        {
            if (is_obstacle(ew) && ex == nx && ez == nz)
            {
                result = MAX(result, ey);
            }
        }
        END_MAP_FOR_EACH;
    }
    return result;
}

void world_resolve_player_collision(const WorldQuery *world, Player *player, PlayerCollisionInput *collision_input, double dt) {
    if(!world || !player || !collision_input) {
        return;
    }
    State *s = &player->state;
    int estimate = roundf(sqrtf(
                              powf(collision_input->vx * collision_input->speed, 2) +
                              powf(collision_input->vy * collision_input->speed + ABS(player->vertical_velocity) * 2, 2) +
                              powf(collision_input->vz * collision_input->speed, 2)) *
                          dt * 8);
    int step = MAX(8, estimate);
    float ut = dt / step;
    collision_input->vx = collision_input->vx * ut * collision_input->speed;
    collision_input->vy = collision_input->vy * ut * collision_input->speed;
    collision_input->vz = collision_input->vz * ut * collision_input->speed;
    for (int i = 0; i < step; i++)
    {
        if (player->flying)
        {
            player->vertical_velocity = 0;
        }
        else
        {
            player->vertical_velocity -= ut * 25;
            player->vertical_velocity = MAX(player->vertical_velocity, -250);
        }
        s->x += collision_input->vx;
        s->y += collision_input->vy + player->vertical_velocity * ut;
        s->z += collision_input->vz;
        if (collide(world, 2, &s->x, &s->y, &s->z))
        {
            player->vertical_velocity = 0;
        }
    }
    if (s->y < 0)
    {
        s->y = world_get_highest_block(world, s->x, s->z) + 2;
    }
}
int world_is_chunk_visible(const Camera *camera_view, int p, int q, int miny, int maxy) {
    int x = p * CHUNK_SIZE - 1;
    int z = q * CHUNK_SIZE - 1;
    int d = CHUNK_SIZE + 1;
    float points[8][3] = {
        {x + 0, miny, z + 0},
        {x + d, miny, z + 0},
        {x + 0, miny, z + d},
        {x + d, miny, z + d},
        {x + 0, maxy, z + 0},
        {x + d, maxy, z + 0},
        {x + 0, maxy, z + d},
        {x + d, maxy, z + d}
    };
    int n = camera_view->ortho ? 4 : 6;
    for (int i = 0; i < n; i++) {
        int in = 0;
        int out = 0;
        for (int j = 0; j < 8; j++) {
            float d =
                camera_view->frustum_planes[i][0] * points[j][0] +
                camera_view->frustum_planes[i][1] * points[j][1] +
                camera_view->frustum_planes[i][2] * points[j][2] +
                camera_view->frustum_planes[i][3];
            if (d < 0) {
                out++;
            }
            else {
                in++;
            }
            if (in && out) {
                break;
            }
        }
        if (in == 0) {
            return 0;
        }
    }
    return 1;
}
int world_hit_test(
    const WorldQuery *world, int previous, float x, float y, float z, float rx, float ry,
    int *bx, int *by, int *bz) {
    
    int result = 0;
    float best = 0;
    int p = chunked(x);
    int q = chunked(z);
    float vx, vy, vz;
    get_sight_vector(rx, ry, &vx, &vy, &vz);
    ChunkIterator iterator = chunk_manager_iterator_begin(world->world_context);
    while(chunk_manager_iterator_has_next(&iterator))
    {
        Chunk *chunk = chunk_manager_iterator_next(&iterator);
        if (chebyshev_distance(chunk->p, chunk->q, p, q) > 1)
        {
            continue;
        }
        int hx, hy, hz;
        int hw = _hit_test(&chunk->map, 8, previous,
                           x, y, z, vx, vy, vz, &hx, &hy, &hz);
        if (hw > 0)
        {
            float d = sqrtf(
                powf(hx - x, 2) + powf(hy - y, 2) + powf(hz - z, 2));
            if (best == 0 || d < best)
            {
                best = d;
                *bx = hx;
                *by = hy;
                *bz = hz;
                result = hw;
            }
        }
    }
    return result;
}
int world_get_block(const WorldQuery *world, int x, int y, int z)
{
    int p = chunked(x);
    int q = chunked(z);
    Chunk *chunk = chunk_manager_find_chunk(world->world_context, p, q);
    if (chunk)
    {
        Map *map = &chunk->map;
        return map_get(map, x, y, z);
    }
    return 0;
}

// INTERNAL HELPERS IMPLEMENTATIONS //
static int collide(const WorldQuery *world, int height, float *x, float *y, float *z) {
    int result = 0;
    int p = chunked(*x);
    int q = chunked(*z);
    Chunk *chunk = chunk_manager_find_chunk(world->world_context, p, q);
    if (!chunk)
    {
        return result;
    }
    Map *map = &chunk->map;
    int nx = roundf(*x);
    int ny = roundf(*y);
    int nz = roundf(*z);
    float px = *x - nx;
    float py = *y - ny;
    float pz = *z - nz;
    float pad = 0.25;
    for (int dy = 0; dy < height; dy++)
    {
        if (px < -pad && is_obstacle(map_get(map, nx - 1, ny - dy, nz)))
        {
            *x = nx - pad;
        }
        if (px > pad && is_obstacle(map_get(map, nx + 1, ny - dy, nz)))
        {
            *x = nx + pad;
        }
        if (py < -pad && is_obstacle(map_get(map, nx, ny - dy - 1, nz)))
        {
            *y = ny - pad;
            result = 1;
        }
        if (py > pad && is_obstacle(map_get(map, nx, ny - dy + 1, nz)))
        {
            *y = ny + pad;
            result = 1;
        }
        if (pz < -pad && is_obstacle(map_get(map, nx, ny - dy, nz - 1)))
        {
            *z = nz - pad;
        }
        if (pz > pad && is_obstacle(map_get(map, nx, ny - dy, nz + 1)))
        {
            *z = nz + pad;
        }
    }
    return result;
}
static void get_sight_vector(float rx, float ry, float *vx, float *vy, float *vz) {
    float m = cosf(ry);
    *vx = cosf(rx - RADIANS(90)) * m;
    *vy = sinf(ry);
    *vz = sinf(rx - RADIANS(90)) * m;
}
static int _hit_test(
    Map *map, float max_distance, int previous,
    float x, float y, float z,
    float vx, float vy, float vz,
    int *hx, int *hy, int *hz)
{
    int m = 32;
    int px = 0;
    int py = 0;
    int pz = 0;
    for (int i = 0; i < max_distance * m; i++)
    {
        int nx = roundf(x);
        int ny = roundf(y);
        int nz = roundf(z);
        if (nx != px || ny != py || nz != pz)
        {
            int hw = map_get(map, nx, ny, nz);
            if (hw > 0)
            {
                if (previous)
                {
                    *hx = px;
                    *hy = py;
                    *hz = pz;
                }
                else
                {
                    *hx = nx;
                    *hy = ny;
                    *hz = nz;
                }
                return hw;
            }
            px = nx;
            py = ny;
            pz = nz;
        }
        x += vx / m;
        y += vy / m;
        z += vz / m;
    }
    return 0;
}