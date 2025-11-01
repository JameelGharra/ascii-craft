#ifndef _chunk_h_
#define _chunk_h_

#include "map.h"
#include "sign.h"
#include <stdint.h>
#include "renderer.h"

typedef struct {
    Map map; // (x, y, z) -> block type
    Map lights; // (x, y, z) -> light level
    SignList signs;
    int p, q; // acts as address of the chunk
    int dirty; // for optimization in mesh rebuilding if 1
    int miny, maxy;
    RenderableObjectID render_id;
} Chunk;

#endif