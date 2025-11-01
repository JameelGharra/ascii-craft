#ifndef __world_query_h__
#define __world_query_h__

#include "camera.h"

typedef struct ChunkManager ChunkManager;
typedef struct Player Player;

typedef struct {
    ChunkManager *world_context;
} WorldQuery;

typedef struct {
    float speed;
    float vx, vy, vz;
} PlayerCollisionInput;

WorldQuery *world_query_create(ChunkManager *world_context);
void world_query_free(WorldQuery *world_query);

int world_get_highest_block(const WorldQuery *world, float x, float z);
void world_resolve_player_collision(const WorldQuery *world, Player *player, PlayerCollisionInput *collision_input, double dt);
int world_is_chunk_visible(const Camera *camera_view, int p, int q, int miny, int maxy);
int world_hit_test(
    const WorldQuery *world, int previous, float x, float y, float z, float rx, float ry,
    int *bx, int *by, int *bz);
int world_get_block(const WorldQuery *world, int x, int y, int z);


#endif