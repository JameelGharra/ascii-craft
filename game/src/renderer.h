#ifndef _renderer_h_
#define _renderer_h_

#include <stdint.h>
#include "config.h"
#include "mesher.h"
#include "window.h"
#include "camera.h"

typedef unsigned int RenderableObjectID; // opaque ID for renderer data
typedef struct Player Player;

#define INVALID_RENDERABLE_OBJECT_ID ((RenderableObjectID)(-1))

typedef struct Attrib Attrib;
typedef struct RenderableChunk RenderableChunk;
typedef struct Renderer Renderer;

void renderer_global_init(); // not per-renderer, just once
Renderer *renderer_create(Window *window);
void renderer_reset(Renderer *renderer);
void renderer_destroy(Renderer **renderer);

void renderer_begin_frame(Renderer *renderer);
void renderer_begin_world_pass(Renderer *renderer); // for the sky depth trick
void renderer_generate_sky_buffer(Renderer *renderer); // a renderer might or might not call this
void renderer_upload_chunk_geometry(Renderer *renderer, RenderableObjectID *id_ptr, MesherOutput *mesh_data);
void renderer_delete_chunk_geometry(Renderer *renderer, RenderableObjectID id);
void renderer_update_player(Renderer *renderer, Player *player);
void renderer_delete_player_geometry(Renderer *renderer, Player *player);

void renderer_render_sky(Renderer *renderer, const Camera *camera_view, float time_of_day);
void renderer_begin_chunk_pass(Renderer *renderer, const Camera *camera_view, float light, float time_of_day);
void renderer_draw_chunk(Renderer *renderer, RenderableObjectID id);
void renderer_begin_sign_pass(Renderer *renderer, const Camera *camera_view);
void renderer_draw_signs(Renderer *renderer, RenderableObjectID id);
void renderer_draw_wireframe(Renderer *renderer, const Camera *camera_view, 
    int hx, int hy, int hz);
void renderer_draw_crosshairs(Renderer *renderer, const Camera *camera_view);
void renderer_begin_item_pass(Renderer *renderer, const Camera *camera_view, float time_of_day);
void renderer_draw_plant(Renderer *renderer, int plant_type);
void renderer_draw_cube(Renderer *renderer, int cube_type);
void renderer_begin_hud_pass(Renderer *renderer);

void renderer_load_textures(Renderer *renderer);
void renderer_load_block_shaders(Renderer *renderer);
void renderer_load_line_shaders(Renderer *renderer);
void renderer_load_text_shaders(Renderer *renderer);
void renderer_load_sky_shaders(Renderer *renderer);

#endif