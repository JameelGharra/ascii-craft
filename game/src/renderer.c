#include <stdlib.h>
#include <string.h>
#include "renderer.h"
#include <GL/glew.h>
#include "player.h"
#include "cube.h"
#include "matrix.h"
#include "util.h"

struct Attrib {
    GLuint program;
    GLuint position;
    GLuint normal;
    GLuint uv;
    GLuint matrix;
    GLuint sampler;
    GLuint camera;
    GLuint timer;
    GLuint extra1;
    GLuint extra2;
    GLuint extra3;
    GLuint extra4;
};
struct RenderableChunk {
    int faces; // faces describe the amount of vertices to be drawn
    int sign_faces;
    GLuint buffer;
    GLuint sign_buffer;
};

struct Renderer {
    Window *window;
    int renderable_chunk_count;
    RenderableChunk renderable_chunks[MAX_CHUNKS];
    GLuint sky_buffer;
    Attrib block_attrib;
    Attrib line_attrib;
    Attrib text_attrib;
    Attrib sky_attrib;
    // at-time binding
    GLuint texture_atlas;
    GLuint font_texture;
    GLuint sky_texture;
    GLuint sign_texture;
};

// INTERNAL HELPERS //
static GLuint _generate_buffer(GLsizei size, GLfloat *data);
static void _delete_buffer(GLuint buffer);
static GLuint _gen_faces(int components, int faces, GLfloat *data);
static GLuint _generate_player_buffer(float x, float y, float z, float rx, float ry);
static void _draw_triangles_3d(Attrib *attrib, GLuint buffer, int count);
static void _draw_item(Attrib *attrib, GLuint buffer, int count);
static GLuint _generate_plant_buffer(float x, float y, float z, float n, int w);
static GLuint _generate_cube_buffer(float x, float y, float z, float n, int w);
static GLuint _generate_crosshair_buffer(const Camera *camera_view, int scale);
static GLuint _generate_wireframe_buffer(float x, float y, float z, float n);
static void _draw_lines(Attrib *attrib, GLuint buffer, int components, int count);
static void _draw_triangles_3d_ao(Attrib *attrib, GLuint buffer, int count);
static void _draw_triangles_3d_text(Attrib *attrib, GLuint buffer, int count);
// ========

void renderer_global_init() {
    glEnable(GL_CULL_FACE);
    glEnable(GL_DEPTH_TEST);
    glLogicOp(GL_INVERT);
    glClearColor(0, 0, 0, 1);
}
Renderer *renderer_create(Window *window) {
    Renderer *renderer = (Renderer *)malloc(sizeof(Renderer));
    if (!renderer) {
        return NULL;
    }
    renderer->sky_buffer = 0;
    renderer->window = window;
    return renderer;
}
void renderer_reset(Renderer *renderer) {
    memset(renderer->renderable_chunks, 0, sizeof(RenderableChunk) * MAX_CHUNKS);
    renderer->renderable_chunk_count = 0;
}
void renderer_destroy(Renderer **renderer) {
    if (renderer && *renderer) {
        if( (*renderer)->sky_buffer ) {
            _delete_buffer((*renderer)->sky_buffer);
        }
        free(*renderer);
        *renderer = NULL;
    }
}
void renderer_begin_frame(Renderer *renderer) {
    int window_width, window_height;
    window_get_scale_factor(renderer->window); // no need to store scale
    window_update_size(renderer->window);
    window_get_size(renderer->window, &window_width, &window_height);
    glViewport(0, 0, window_width, window_height);
    glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
}
void renderer_begin_world_pass(Renderer *renderer) {
    glClear(GL_DEPTH_BUFFER_BIT);
}
void renderer_delete_chunk_geometry(Renderer *renderer, RenderableObjectID id) {
    if(id < 0 || id >= MAX_CHUNKS) {
        return;
    }
    RenderableChunk *chunk = &renderer->renderable_chunks[id];
    _delete_buffer(chunk->buffer);
    _delete_buffer(chunk->sign_buffer);
    RenderableChunk *other = &renderer->renderable_chunks[--renderer->renderable_chunk_count];
    memcpy(chunk, other, sizeof(RenderableChunk));
}
void renderer_upload_chunk_geometry(Renderer *renderer, RenderableObjectID *id_ptr, MesherOutput *mesh_data) {
    if(!id_ptr || !mesh_data || *id_ptr >= MAX_CHUNKS) {
        return;
    }
    if(*id_ptr != INVALID_RENDERABLE_OBJECT_ID) {
        _delete_buffer(renderer->renderable_chunks[*id_ptr].buffer);
        _delete_buffer(renderer->renderable_chunks[*id_ptr].sign_buffer);
    }
    else {
        *id_ptr = renderer->renderable_chunk_count++;
    }
    RenderableChunk *chunk = &renderer->renderable_chunks[*id_ptr];
    chunk->faces = mesh_data->faces;
    chunk->sign_faces = mesh_data->sign_faces;
    chunk->buffer = _gen_faces(10, mesh_data->faces, mesh_data->data);
    chunk->sign_buffer = _gen_faces(5, chunk->sign_faces, mesh_data->sign_data);
}
void renderer_update_player(Renderer *renderer, Player *player) {
    if(!renderer || !player) {
        return;
    }
    if(player->render_id != INVALID_RENDERABLE_OBJECT_ID) { // in an ideal world, wont be needed
        _delete_buffer(player->render_id);
    }
    player->render_id = _generate_player_buffer(
        player->state.x, player->state.y, player->state.z,
        player->state.rx, player->state.ry);
}
void renderer_delete_player_geometry(Renderer *renderer, Player *player) {
    if(!renderer || !player || player->render_id == INVALID_RENDERABLE_OBJECT_ID) {
        return;
    }
    _delete_buffer(player->render_id);
    player->render_id = INVALID_RENDERABLE_OBJECT_ID;
}
void renderer_render_sky(Renderer *renderer, const Camera *camera_view, float time_of_day) {
    float matrix[16];
    // created the matrix in here on purpose, irregular parameters with 0
    set_matrix_3d(
        matrix, camera_view->window_width, camera_view->window_height,
        0, 0, 0, camera_view->rx, camera_view->ry, camera_view->fov, 0, camera_view->render_radius);
    glUseProgram(renderer->sky_attrib.program);
    glUniformMatrix4fv(renderer->sky_attrib.matrix, 1, GL_FALSE, matrix);
    glUniform1i(renderer->sky_attrib.sampler, 2);
    glUniform1f(renderer->sky_attrib.timer, time_of_day);
    _draw_triangles_3d(&renderer->sky_attrib, renderer->sky_buffer, 512 * 3);
}
void renderer_begin_chunk_pass(Renderer *renderer, const Camera *camera_view, float light, float time_of_day) {
    glUseProgram(renderer->block_attrib.program);
    glUniformMatrix4fv(renderer->block_attrib.matrix, 1, GL_FALSE, camera_view->view_proj_matrix);
    glUniform3f(renderer->block_attrib.camera, camera_view->x, camera_view->y, camera_view->z);
    glUniform1i(renderer->block_attrib.sampler, 0);
    glUniform1i(renderer->block_attrib.extra1, 2);
    glUniform1f(renderer->block_attrib.extra2, light);
    glUniform1f(renderer->block_attrib.extra3, camera_view->render_radius * CHUNK_SIZE);
    glUniform1i(renderer->block_attrib.extra4, camera_view->ortho);
    glUniform1f(renderer->block_attrib.timer, time_of_day);
}
void renderer_draw_chunk(Renderer *renderer, RenderableObjectID id) {
    if(id < 0 || id >= MAX_CHUNKS) {
        return;
    }
    RenderableChunk *chunk = &renderer->renderable_chunks[id];
    _draw_triangles_3d_ao(&renderer->block_attrib, chunk->buffer, chunk->faces * 6);
}
void renderer_begin_sign_pass(Renderer *renderer, const Camera *camera_view) {
    glUseProgram(renderer->text_attrib.program);
    glUniformMatrix4fv(renderer->text_attrib.matrix, 1, GL_FALSE, camera_view->view_proj_matrix);
    glUniform1i(renderer->text_attrib.sampler, 3);
    glUniform1i(renderer->text_attrib.extra1, 1);
}
void renderer_draw_signs(Renderer *renderer, RenderableObjectID id) {
    if(id < 0 || id >= MAX_CHUNKS) {
        return;
    }
    RenderableChunk *chunk = &renderer->renderable_chunks[id];
    glEnable(GL_POLYGON_OFFSET_FILL);
    glPolygonOffset(-8, -1024);
    _draw_triangles_3d_text(&renderer->text_attrib, chunk->sign_buffer, chunk->sign_faces * 6);
    glDisable(GL_POLYGON_OFFSET_FILL);
}
void renderer_draw_wireframe(Renderer *renderer, const Camera *camera_view, 
    int hx, int hy, int hz) {

    glUseProgram(renderer->line_attrib.program);
    glLineWidth(1);
    glEnable(GL_COLOR_LOGIC_OP);
    glUniformMatrix4fv(renderer->line_attrib.matrix, 1, GL_FALSE, camera_view->view_proj_matrix);
    GLuint wireframe_buffer = _generate_wireframe_buffer(hx, hy, hz, 0.53);
    _draw_lines(&renderer->line_attrib, wireframe_buffer, 3, 24);
    _delete_buffer(wireframe_buffer);
    glDisable(GL_COLOR_LOGIC_OP);
}
void renderer_draw_crosshairs(Renderer *renderer, const Camera *camera_view) {
    float matrix[16];
    set_matrix_2d(matrix, camera_view->window_width, camera_view->window_height);
    glUseProgram(renderer->line_attrib.program);
    int scale = window_get_scale_factor(renderer->window);
    glLineWidth(4 * scale);
    glEnable(GL_COLOR_LOGIC_OP);
    glUniformMatrix4fv(renderer->line_attrib.matrix, 1, GL_FALSE, matrix);
    GLuint crosshair_buffer = _generate_crosshair_buffer(camera_view, scale);
    _draw_lines(&renderer->line_attrib, crosshair_buffer, 2, 4);
    _delete_buffer(crosshair_buffer);
    glDisable(GL_COLOR_LOGIC_OP);
}
void renderer_begin_item_pass(Renderer *renderer, const Camera *camera_view, float time_of_day) {
    float matrix[16];
    int scale = window_get_scale_factor(renderer->window);
    set_matrix_item(matrix, camera_view->window_width, camera_view->window_height, scale);
    glUseProgram(renderer->block_attrib.program);
    glUniformMatrix4fv(renderer->block_attrib.matrix, 1, GL_FALSE, matrix);
    glUniform3f(renderer->block_attrib.camera, 0, 0, 5);
    glUniform1i(renderer->block_attrib.sampler, 0);
    glUniform1f(renderer->block_attrib.timer, time_of_day);
}
void renderer_draw_plant(Renderer *renderer, int plant_type) {
    GLuint buffer = _generate_plant_buffer(0, 0, 0, 0.5, plant_type);
    _draw_item(&renderer->block_attrib, buffer, 24);
    _delete_buffer(buffer);
}
void renderer_draw_cube(Renderer *renderer, int cube_type) {
    GLuint buffer = _generate_cube_buffer(0, 0, 0, 0.5, cube_type);
    _draw_item(&renderer->block_attrib, buffer, 36);
    _delete_buffer(buffer);
}
void renderer_begin_hud_pass(Renderer *renderer) {
    glClear(GL_DEPTH_BUFFER_BIT);
}
void renderer_generate_sky_buffer(Renderer *renderer) {
    float data[12288];
    make_sphere(data, 1, 3);
    renderer->sky_buffer = _generate_buffer(sizeof(data), data);
}
void renderer_load_textures(Renderer *renderer) {
    GLuint texture;
    glGenTextures(1, &texture);
    glActiveTexture(GL_TEXTURE0);
    glBindTexture(GL_TEXTURE_2D, texture);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
    load_png_texture("textures/texture.png");
    renderer->texture_atlas = texture;

    GLuint font;
    glGenTextures(1, &font);
    glActiveTexture(GL_TEXTURE1);
    glBindTexture(GL_TEXTURE_2D, font);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
    load_png_texture("textures/font.png");
    renderer->font_texture = font;

    GLuint sky;
    glGenTextures(1, &sky);
    glActiveTexture(GL_TEXTURE2);
    glBindTexture(GL_TEXTURE_2D, sky);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, GL_CLAMP_TO_EDGE);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, GL_CLAMP_TO_EDGE);
    load_png_texture("textures/sky.png");
    renderer->sky_texture = sky;

    GLuint sign;
    glGenTextures(1, &sign);
    glActiveTexture(GL_TEXTURE3);
    glBindTexture(GL_TEXTURE_2D, sign);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_NEAREST);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_NEAREST);
    load_png_texture("textures/sign.png");
    renderer->sign_texture = sign;
}
void renderer_load_block_shaders(Renderer *renderer) {
    renderer->block_attrib = (Attrib){0};
    GLuint program = load_program(
        "shaders/block_vertex.glsl", "shaders/block_fragment.glsl");
    renderer->block_attrib.program = program;
    renderer->block_attrib.position = glGetAttribLocation(program, "position");
    renderer->block_attrib.normal = glGetAttribLocation(program, "normal");
    renderer->block_attrib.uv = glGetAttribLocation(program, "uv");
    renderer->block_attrib.matrix = glGetUniformLocation(program, "matrix");
    renderer->block_attrib.sampler = glGetUniformLocation(program, "sampler");
    renderer->block_attrib.extra1 = glGetUniformLocation(program, "sky_sampler");
    renderer->block_attrib.extra2 = glGetUniformLocation(program, "daylight");
    renderer->block_attrib.extra3 = glGetUniformLocation(program, "fog_distance");
    renderer->block_attrib.extra4 = glGetUniformLocation(program, "ortho");
    renderer->block_attrib.camera = glGetUniformLocation(program, "camera");
    renderer->block_attrib.timer = glGetUniformLocation(program, "timer");
}
void renderer_load_line_shaders(Renderer *renderer) {
    renderer->line_attrib = (Attrib){0};
    GLuint program = load_program(
        "shaders/line_vertex.glsl", "shaders/line_fragment.glsl");
    renderer->line_attrib.program = program;
    renderer->line_attrib.position = glGetAttribLocation(program, "position");
    renderer->line_attrib.matrix = glGetUniformLocation(program, "matrix");
}
void renderer_load_text_shaders(Renderer *renderer) {
    renderer->text_attrib = (Attrib){0};
    GLuint program = load_program(
        "shaders/text_vertex.glsl", "shaders/text_fragment.glsl");
    renderer->text_attrib.program = program;
    renderer->text_attrib.position = glGetAttribLocation(program, "position");
    renderer->text_attrib.uv = glGetAttribLocation(program, "uv");
    renderer->text_attrib.matrix = glGetUniformLocation(program, "matrix");
    renderer->text_attrib.sampler = glGetUniformLocation(program, "sampler");
    renderer->text_attrib.extra1 = glGetUniformLocation(program, "is_sign");
}
void renderer_load_sky_shaders(Renderer *renderer) {
    renderer->sky_attrib = (Attrib){0};
    GLuint program = load_program(
        "shaders/sky_vertex.glsl", "shaders/sky_fragment.glsl");
    renderer->sky_attrib.program = program;
    renderer->sky_attrib.position = glGetAttribLocation(program, "position");
    renderer->sky_attrib.normal = glGetAttribLocation(program, "normal");
    renderer->sky_attrib.uv = glGetAttribLocation(program, "uv");
    renderer->sky_attrib.matrix = glGetUniformLocation(program, "matrix");
    renderer->sky_attrib.sampler = glGetUniformLocation(program, "sampler");
    renderer->sky_attrib.timer = glGetUniformLocation(program, "timer");
}

// INTERNAL HELPERS IMPLEMENTATIONS //

static GLuint _generate_buffer(GLsizei size, GLfloat *data) {
    GLuint buffer;
    glGenBuffers(1, &buffer);
    glBindBuffer(GL_ARRAY_BUFFER, buffer);
    glBufferData(GL_ARRAY_BUFFER, size, data, GL_STATIC_DRAW);
    glBindBuffer(GL_ARRAY_BUFFER, 0);
    return buffer;
}
static void _delete_buffer(GLuint buffer) {
    glDeleteBuffers(1, &buffer);
}
static GLuint _gen_faces(int components, int faces, GLfloat *data) {
    GLuint buffer = _generate_buffer(
        sizeof(GLfloat) * 6 * components * faces, data);
    return buffer;
}
static GLuint _generate_player_buffer(float x, float y, float z, float rx, float ry) {
    GLfloat *data = malloc_faces(10, 6);
    make_player(data, x, y, z, rx, ry);
    return _gen_faces(10, 6, data);
}
static void _draw_triangles_3d(Attrib *attrib, GLuint buffer, int count) {
    glBindBuffer(GL_ARRAY_BUFFER, buffer);
    glEnableVertexAttribArray(attrib->position);
    glEnableVertexAttribArray(attrib->normal);
    glEnableVertexAttribArray(attrib->uv);
    glVertexAttribPointer(attrib->position, 3, GL_FLOAT, GL_FALSE,
                          sizeof(GLfloat) * 8, 0);
    glVertexAttribPointer(attrib->normal, 3, GL_FLOAT, GL_FALSE,
                          sizeof(GLfloat) * 8, (GLvoid *)(sizeof(GLfloat) * 3));
    glVertexAttribPointer(attrib->uv, 2, GL_FLOAT, GL_FALSE,
                          sizeof(GLfloat) * 8, (GLvoid *)(sizeof(GLfloat) * 6));
    glDrawArrays(GL_TRIANGLES, 0, count);
    glDisableVertexAttribArray(attrib->position);
    glDisableVertexAttribArray(attrib->normal);
    glDisableVertexAttribArray(attrib->uv);
    glBindBuffer(GL_ARRAY_BUFFER, 0);
}
static void _draw_item(Attrib *attrib, GLuint buffer, int count) {
    _draw_triangles_3d_ao(attrib, buffer, count);
}
static GLuint _generate_plant_buffer(float x, float y, float z, float n, int w) {
    GLfloat *data = malloc_faces(10, 4);
    float ao = 0;
    float light = 1;
    make_plant(data, ao, light, x, y, z, n, w, 45);
    return _gen_faces(10, 4, data);
}
static GLuint _generate_cube_buffer(float x, float y, float z, float n, int w) {
    GLfloat *data = malloc_faces(10, 6);
    float ao[6][4] = {0};
    float light[6][4] = {
        {0.5, 0.5, 0.5, 0.5},
        {0.5, 0.5, 0.5, 0.5},
        {0.5, 0.5, 0.5, 0.5},
        {0.5, 0.5, 0.5, 0.5},
        {0.5, 0.5, 0.5, 0.5},
        {0.5, 0.5, 0.5, 0.5}};
    make_cube(data, ao, light, 1, 1, 1, 1, 1, 1, x, y, z, n, w);
    return _gen_faces(10, 6, data);
}
static GLuint _generate_crosshair_buffer(const Camera *camera_view, int scale) {
    int x = camera_view->window_width / 2;
    int y = camera_view->window_height / 2;
    int p = 10 * scale;
    float data[] = {
        x, y - p, x, y + p,
        x - p, y, x + p, y};
    return _generate_buffer(sizeof(data), data);
}
static GLuint _generate_wireframe_buffer(float x, float y, float z, float n) {
    float data[72];
    make_cube_wireframe(data, x, y, z, n);
    return _generate_buffer(sizeof(data), data);
}
static void _draw_lines(Attrib *attrib, GLuint buffer, int components, int count) {
    glBindBuffer(GL_ARRAY_BUFFER, buffer);
    glEnableVertexAttribArray(attrib->position);
    glVertexAttribPointer(
        attrib->position, components, GL_FLOAT, GL_FALSE, 0, 0);
    glDrawArrays(GL_LINES, 0, count);
    glDisableVertexAttribArray(attrib->position);
    glBindBuffer(GL_ARRAY_BUFFER, 0);
}
static void _draw_triangles_3d_ao(Attrib *attrib, GLuint buffer, int count) {
    glBindBuffer(GL_ARRAY_BUFFER, buffer);
    glEnableVertexAttribArray(attrib->position);
    glEnableVertexAttribArray(attrib->normal);
    glEnableVertexAttribArray(attrib->uv);
    glVertexAttribPointer(attrib->position, 3, GL_FLOAT, GL_FALSE,
                          sizeof(GLfloat) * 10, 0);
    glVertexAttribPointer(attrib->normal, 3, GL_FLOAT, GL_FALSE,
                          sizeof(GLfloat) * 10, (GLvoid *)(sizeof(GLfloat) * 3));
    glVertexAttribPointer(attrib->uv, 4, GL_FLOAT, GL_FALSE,
                          sizeof(GLfloat) * 10, (GLvoid *)(sizeof(GLfloat) * 6));
    glDrawArrays(GL_TRIANGLES, 0, count);
    glDisableVertexAttribArray(attrib->position);
    glDisableVertexAttribArray(attrib->normal);
    glDisableVertexAttribArray(attrib->uv);
    glBindBuffer(GL_ARRAY_BUFFER, 0);
}
static void _draw_triangles_3d_text(Attrib *attrib, GLuint buffer, int count) {
    glBindBuffer(GL_ARRAY_BUFFER, buffer);
    glEnableVertexAttribArray(attrib->position);
    glEnableVertexAttribArray(attrib->uv);
    glVertexAttribPointer(attrib->position, 3, GL_FLOAT, GL_FALSE,
                          sizeof(GLfloat) * 5, 0);
    glVertexAttribPointer(attrib->uv, 2, GL_FLOAT, GL_FALSE,
                          sizeof(GLfloat) * 5, (GLvoid *)(sizeof(GLfloat) * 3));
    glDrawArrays(GL_TRIANGLES, 0, count);
    glDisableVertexAttribArray(attrib->position);
    glDisableVertexAttribArray(attrib->uv);
    glBindBuffer(GL_ARRAY_BUFFER, 0);
}
