#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include "config.h"
#include "window.h"
#include "db.h"
#include "item.h"
#include "map.h"
#include "util.h"
#include "player.h"
#include "renderer.h"
#include "camera.h"
#include "chunk_manager.h"
#include "input_manager.h"
#include "world_query.h"
#include "time.h"
#include "game_clock.h"
#include "ipc.h"

#if ASCII_MODE
#include "ascii_renderer.h"
#endif
#include "automation_bot.h"

#define MAX_PATH_LENGTH 256

typedef struct
{
    Window *window;
    Renderer *renderer;
    ChunkManager *chunk_manager;
    InputManager *input_manager;
    Player local_player;
    int item_index;
    int ortho;
    float fov;
    char db_path[MAX_PATH_LENGTH];
    int day_length;
    int time_changed;
    #if ASCII_MODE
        AsciiRenderer *ascii_renderer;
    #endif
    #if AUTOMATION_BOT_MODE
        AutomationBot *automation_bot;
    #endif
} Model;

static Model model;
static Model *g = &model;


void proceed_render_chunks(const Camera *view) {
    int p = chunked(view->x);
    int q = chunked(view->z);
    float light = game_clock_get_daylight();
    renderer_begin_chunk_pass(g->renderer, view, light, game_clock_get_time_of_day());
    ChunkIterator it = chunk_manager_iterator_begin(g->chunk_manager);
    while(chunk_manager_iterator_has_next(&it)) {
        Chunk *chunk = chunk_manager_iterator_next(&it);
        if (chebyshev_distance(chunk->p, chunk->q, p, q) > view->render_radius)
        {
            continue;
        }
        if (!world_is_chunk_visible(
            view, chunk->p, chunk->q, chunk->miny, chunk->maxy))
        {
            continue;
        }
        renderer_draw_chunk(g->renderer, chunk->render_id);
    }
}
void proceed_render_signs(const Camera *view) {
    int p = chunked(view->x);
    int q = chunked(view->z);
    renderer_begin_sign_pass(g->renderer, view);
    ChunkIterator it = chunk_manager_iterator_begin(g->chunk_manager);
    while(chunk_manager_iterator_has_next(&it)) {
        Chunk *chunk = chunk_manager_iterator_next(&it);
        if (chebyshev_distance(chunk->p, chunk->q, p, q) > RENDER_SIGN_RADIUS)
        {
            continue;
        }
        if (!world_is_chunk_visible(
            view, chunk->p, chunk->q, chunk->miny, chunk->maxy))
        {
            continue;
        }
        renderer_draw_signs(g->renderer, chunk->render_id);
    }    
}

void proceed_render_wireframe(const Camera *view, WorldQuery *world)
{
    int hx, hy, hz;
    int hw = world_hit_test(world, 0, view->x, view->y, view->z, view->rx, view->ry, &hx, &hy, &hz);
    if (is_obstacle(hw)) {
        renderer_draw_wireframe(g->renderer, view, hx, hy, hz);
    }
}

void proceed_render_item(const Camera *view) {

    renderer_begin_item_pass(g->renderer, view, game_clock_get_time_of_day());
    int w = items[g->item_index];
    if (is_plant(w)) {
        renderer_draw_plant(g->renderer, w);
    }
    else {
        renderer_draw_cube(g->renderer, w);
    }
}
void on_light(const WorldQuery *world_query) {
    State *s = &g->local_player.state;
    int hx, hy, hz;
    int hw = world_hit_test(world_query, 0, s->x, s->y, s->z, s->rx, s->ry, &hx, &hy, &hz);
    if (hy > 0 && hy < 256 && is_destructable(hw)) {
        chunk_manager_toggle_light(g->chunk_manager, hx, hy, hz);
    }
}

void on_destroy(const WorldQuery *world_query)
{
    State *s = &g->local_player.state;
    int hx, hy, hz;
    int hw = world_hit_test(world_query, 0, s->x, s->y, s->z, s->rx, s->ry, &hx, &hy, &hz);
    if (hy > 0 && hy < 256 && is_destructable(hw))
    {
        chunk_manager_set_block(g->chunk_manager, hx, hy, hz, 0);
        if (is_plant(world_get_block(world_query, hx, hy + 1, hz)))
        {
            chunk_manager_set_block(g->chunk_manager, hx, hy + 1, hz, 0);
        }
    }
}

void on_build(const WorldQuery *world_query) {
    State *s = &g->local_player.state;
    int hx, hy, hz;
    int hw = world_hit_test(world_query, 1, s->x, s->y, s->z, s->rx, s->ry, &hx, &hy, &hz);
    if (hy > 0 && hy < 256 && is_obstacle(hw))
    {
        if (!is_point_in_vertical_bounds(2, s->x, s->y, s->z, hx, hy, hz))
        {
            chunk_manager_set_block(g->chunk_manager, hx, hy, hz, items[g->item_index]);
        }
    }
}

void on_copy(const WorldQuery *world_query) {
    State *s = &g->local_player.state;
    int hx, hy, hz;
    int hw = world_hit_test(world_query, 0, s->x, s->y, s->z, s->rx, s->ry, &hx, &hy, &hz);
    for (int i = 0; i < item_count; i++)
    {
        if (items[i] == hw)
        {
            g->item_index = i;
            break;
        }
    }
}

void on_scroll(double y_delta) {
    static double y_pos = 0;
    y_pos += y_delta;
    if (y_pos < -SCROLL_THRESHOLD) {
        g->item_index = (g->item_index + 1) % item_count;
        y_pos = 0;
    }
    if (y_pos > SCROLL_THRESHOLD) {
        g->item_index--;
        if (g->item_index < 0) {
            g->item_index = item_count - 1;
        }
        y_pos = 0;
    }
}

void handle_mouse_input(double dt) {
    #if AUTOMATION_BOT_MODE
        automation_bot_look_update(g->automation_bot, &g->local_player, dt);
    #endif
    double m_dx, m_dy;
    window_get_cursor_delta(g->window, &m_dx, &m_dy);
    player_update_look(&g->local_player, m_dx, m_dy, MOUSE_SENSITIVITY);
}
void handle_key_movement(Player *me, WorldQuery *world_query, double dt) {
    PlayerMovementIntent intent = {0};
    if (window_is_key_down(g->window, CRAFT_KEY_FORWARD)) {
        intent.forward = true;
    }
    if (window_is_key_down(g->window, CRAFT_KEY_BACKWARD)) {
        intent.backward = true;
    }
    if (window_is_key_down(g->window, CRAFT_KEY_LEFT)) {
        intent.left = true;
    }
    if (window_is_key_down(g->window, CRAFT_KEY_RIGHT)) {
        intent.right = true;
    }
    if (window_is_key_down(g->window, CRAFT_KEY_JUMP)) {
        intent.jump = true;
    }
    if(window_is_key_down(g->window, KEY_LEFT)) {
        intent.turn_left = true;
    }
    if(window_is_key_down(g->window, KEY_RIGHT)) {
        intent.turn_right = true;
    }
    if(window_is_key_down(g->window, KEY_UP)) {
        intent.turn_up = true;
    }
    if(window_is_key_down(g->window, KEY_DOWN)) {
        intent.turn_down = true;
    }
    #if AUTOMATION_BOT_MODE
        automation_bot_pos_update(g->automation_bot, me, &intent, dt);
    #endif
    player_update_pos(me, &intent, world_query, dt);
}
void reset_model()
{
    chunk_manager_reset(g->chunk_manager);
    renderer_reset(g->renderer);
    g->item_index = 0;
    g->time_changed = 1;
    game_clock_set_day_length(DAY_LENGTH);
    game_clock_set_to(TIME_OF_DAY_MORNING);
}
void update_ortho_zoom() {
    g->ortho = window_is_key_down(g->window, CRAFT_KEY_ORTHO) ? 64 : 0;
    g->fov = window_is_key_down(g->window, CRAFT_KEY_ZOOM) ? 15 : 65;
}
void retrieve_camera_view_params(Player *player, Camera *view) {
    State *s = &player->state;
    view->x = s->x;
    view->y = s->y;
    view->z = s->z;
    view->rx = s->rx;
    view->ry = s->ry;
    view->fov = g->fov;
    view->ortho = g->ortho;
    window_get_size(g->window, &view->window_width, &view->window_height);
    view->render_radius = RENDER_CHUNK_RADIUS;
    camera_update_matrices(view);
}
void load_shaders() {
    renderer_load_block_shaders(g->renderer);
    renderer_load_line_shaders(g->renderer);
    renderer_load_text_shaders(g->renderer);
    renderer_load_sky_shaders(g->renderer);
}
int initialize_main_game_core() {
    srand(time(NULL));
    rand();
    time_init();
    game_clock_init(DAY_LENGTH);
    if (!window_global_init())
    {
        return 0;
    }
    if((g->window = window_create(WINDOW_WIDTH, WINDOW_HEIGHT, "Craft", FULLSCREEN)) == NULL) {
        window_terminate();
        return 0;
    }

    renderer_global_init();
    g->renderer = renderer_create(g->window);
    g->chunk_manager = chunk_manager_create(&(ChunkManagerConfig) {
        .create_radius = CREATE_CHUNK_RADIUS,
        .render_radius = RENDER_CHUNK_RADIUS,
        .delete_radius = DELETE_CHUNK_RADIUS,
        .sign_radius = RENDER_SIGN_RADIUS,
    });
    #if ASCII_MODE
        AsciiConfig config = {
            .ascii_width = ASCII_WINDOW_WIDTH,
            .ascii_height = ASCII_WINDOW_HEIGHT,
            .source_width = WINDOW_WIDTH,
            .source_height = WINDOW_HEIGHT
        };
        g->ascii_renderer = ascii_renderer_create(&config);
        ipc_create();
    #endif
    g->input_manager = input_manager_create(g->window);
    if(!g->window || !g->renderer || !g->chunk_manager || !g->input_manager) {
        return 0;
    }
    #if ASCII_MODE
    if(!g->ascii_renderer) {
        return 0;
    }
    #endif
    return 1;
}
void destroy_main_game_core() {
    if (g->chunk_manager) {
        chunk_manager_destroy(g->chunk_manager, g->renderer);
        g->chunk_manager = NULL;
    }
    if (g->renderer) {
        renderer_destroy(&g->renderer);
        g->renderer = NULL;
    }
    if (g->input_manager) {
        input_manager_free(g->input_manager);
        g->input_manager = NULL;
    }
    if (g->window) {
        window_free(g->window);
        g->window = NULL;
    }
    #if ASCII_MODE
        ascii_renderer_destroy(&g->ascii_renderer);
        ipc_destroy();
    #endif
    window_terminate();
}
void handle_commands(const WorldQuery *world_query) {
    GameCommand command;
    while(input_manager_get_next_command(g->input_manager, &command)) {
        switch(command.type) {
            case COMMAND_TOGGLE_FLY: {
                g->local_player.flying = !g->local_player.flying;
                break;
            }
            case COMMAND_BUILD: {
                on_build(world_query);
                break;
            }
            case COMMAND_DESTROY: {
                on_destroy(world_query);
                break;
            }
            case COMMAND_SET_ITEM_INDEX: {
                if(command.data.set_item.index < 0 || command.data.set_item.index >= item_count) {
                    break;
                }
                g->item_index = command.data.set_item.index;
                break;
            }
            case COMMAND_CYCLE_UP_ITEM: {
                g->item_index = (g->item_index + 1) % item_count;
                break;
            }
            case COMMAND_CYCLE_DOWN_ITEM: {
                g->item_index--;
                if (g->item_index < 0) {
                    g->item_index = item_count - 1;
                }
                break;
            }
            case COMMAND_SCROLL: {
                on_scroll(command.data.cycle_item.y_delta);
                break;
            }
            case COMMAND_MIDDLE_COPY_ELEMENT: {
                on_copy(world_query);
                break;
            }
            case COMMAND_TOGGLE_LIGHT: {
                on_light(world_query);
                break;
            }
            #if AUTOMATION_BOT_MODE
            case COMMAND_MOVE_FORWARD: case COMMAND_MOVE_BACKWARD:
            case COMMAND_MOVE_LEFT: case COMMAND_MOVE_RIGHT: {
                g->automation_bot->active_move_dir = command.type - COMMAND_MOVE_FORWARD + 1;
                g->automation_bot->move_timer = command.data.movement.duration;
                g->automation_bot->holding_jump = command.data.movement.jump;
                break;
            }
            case COMMAND_LOOK_YAW: {
                if (!g->automation_bot->is_rotating) { // even if new cmd comes, finish prev and drop the new one
                    g->automation_bot->is_rotating = true;
                    g->automation_bot->rotate_timer = 0;
                    g->automation_bot->rotate_duration = command.data.look.duration;
                    
                    g->automation_bot->start_rx = g->local_player.state.rx;
                    g->automation_bot->start_ry = g->local_player.state.ry;
                    
                    g->automation_bot->target_rx = g->automation_bot->start_rx + RADIANS(command.data.look.angle_delta);
                    g->automation_bot->target_ry = g->automation_bot->start_ry; 
                }
                break;
            }
            case COMMAND_LOOK_PITCH: {
                if (!g->automation_bot->is_rotating) {
                    g->automation_bot->is_rotating = true;
                    g->automation_bot->rotate_timer = 0;
                    g->automation_bot->rotate_duration = command.data.look.duration;
                    
                    g->automation_bot->start_rx = g->local_player.state.rx;
                    g->automation_bot->start_ry = g->local_player.state.ry;
                    
                    // calculate target and clamp pitch to -90/90 (pi/2)
                    float target = g->automation_bot->start_ry + RADIANS(command.data.look.angle_delta);
                    target = MAX(target, -RADIANS(90));
                    target = MIN(target, RADIANS(90));
                    
                    g->automation_bot->target_rx = g->automation_bot->start_rx;
                    g->automation_bot->target_ry = target;
                }
                break;
            }
            case COMMAND_JUMP: {
                g->automation_bot->holding_jump = true;
                g->automation_bot->active_move_dir = MOVE_DIR_NONE;
                g->automation_bot->move_timer = command.data.movement.duration;
                break;
            }
            #endif
        }
    }
}
int main(int argc, char **argv)
{
    // unsigned int frames = 0;
    // INITIALIZATION //
    if(!initialize_main_game_core()) {
        destroy_main_game_core();
        return -1;
    }

    // CHECK COMMAND LINE ARGUMENTS //
    snprintf(g->db_path, MAX_PATH_LENGTH, "%s", DB_PATH);

    // LOAD TEXTURES AND SHADERS //
    renderer_load_textures(g->renderer);
    load_shaders();

    // DATABASE INITIALIZATION //
    db_enable();
    if (db_init(g->db_path))
    {
        return -1;
    }

    // LOCAL VARIABLES //
    reset_model();
    FPS fps = {0, 0, 0};
    double last_commit = time_get_seconds();
    double last_update = time_get_seconds();
    renderer_generate_sky_buffer(g->renderer);

    Player *me = &g->local_player;
    #if AUTOMATION_BOT_MODE
        g->automation_bot = &(AutomationBot){0};
        automation_bot_reset(g->automation_bot);
    #endif
    State *s = &me->state;
    player_init(me);

    // LOAD STATE FROM DATABASE //
    int loaded = db_load_state(&s->x, &s->y, &s->z, &s->rx, &s->ry);
    chunk_manager_force_chunks_around_point(g->chunk_manager, g->renderer, s->x, s->z);
    WorldQuery *world_query = world_query_create(g->chunk_manager);
    if (!loaded) {
        s->y = world_get_highest_block(world_query, s->x, s->z) + 2;
    }

    // BEGIN MAIN LOOP //
    double previous = time_get_seconds();
    while (true) {
        // printf("Frame number: %u\n", frames++);
        // WINDOW SIZE, SCALE AND CLEAR CANVAS //
        #if ASCII_MODE
            ascii_renderer_bind_offscreen_buffer(g->ascii_renderer);
        #endif
        renderer_begin_frame(g->renderer);

        // FRAME RATE //
        if (g->time_changed) {
            g->time_changed = 0;
            last_commit = time_get_seconds();
            last_update = time_get_seconds();
            memset(&fps, 0, sizeof(fps));
        }
        update_fps(&fps);
        double now = time_get_seconds();
        double dt = now - previous;
        dt = MIN(dt, 0.2);
        dt = MAX(dt, 0.0);
        previous = now;

        // ACTIONS //
        update_ortho_zoom();
        input_manager_update(g->input_manager, g->window);
        handle_commands(world_query);
        handle_mouse_input(dt);
        handle_key_movement(me, world_query, dt); // continuous
        
        // FLUSH DATABASE //
        if (now - last_commit > COMMIT_INTERVAL) {
            last_commit = now;
            db_commit();
        }

        // PREPARE TO RENDER //
       chunk_manager_delete_distant_chunks(g->chunk_manager, g->renderer, s->x, s->z);
        renderer_update_player(g->renderer, me);

        // RENDER 3-D SCENE //
        Camera view;
        retrieve_camera_view_params(me, &view);
        renderer_render_sky(g->renderer, &view, game_clock_get_time_of_day());
        renderer_begin_world_pass(g->renderer);
        // printf("Camera Position: (%.2f, %.2f, %.2f)  Rotation: (%.2f, %.2f)  FOV: %.2f  Ortho: %d\n",
            // view.x, view.y, view.z, view.rx, view.ry, view.fov, view.ortho);
        chunk_manager_update(g->chunk_manager, &view, g->renderer);
        // printf("ChunkManager updated the chunks \n");
        proceed_render_chunks(&view);
        // printf("Chunks rendered \n");
        proceed_render_signs(&view);
        // printf("Signs rendered \n");

        if (SHOW_WIREFRAME) {
            proceed_render_wireframe(&view, world_query);
        }

        // RENDER HUD //
        renderer_begin_hud_pass(g->renderer);
        if (SHOW_CROSSHAIRS) {
            renderer_draw_crosshairs(g->renderer, &view);
        }
        if (SHOW_ITEM) {
            proceed_render_item(&view);
        }
        // SWAP AND POLL //
        #if ASCII_MODE
            ascii_renderer_read_pixels(g->ascii_renderer);
            ipc_notify_data_not_ready();
            uint32_t data_len = ascii_renderer_render(g->ascii_renderer, (AsciiPixel*)ipc_get_data_pointer());
            uint32_t ascii_width, ascii_height;
            ascii_renderer_get_target_size(g->ascii_renderer, &ascii_width, &ascii_height);
            ipc_notify_data_ready(ascii_width, ascii_height, data_len);
            #if ASCII_LOCAL_PRINT
                ascii_renderer_print_debug(g->ascii_renderer, (AsciiPixel*)ipc_get_data_pointer());
            #endif
        #endif
        if(!window_next_frame(g->window)) {
            break;
        }
    }
    // SHUTDOWN //
    db_save_state(s->x, s->y, s->z, s->rx, s->ry);
    db_close();
    db_disable();
    world_query_free(world_query);
    renderer_delete_player_geometry(g->renderer, me);
    chunk_manager_destroy(g->chunk_manager, g->renderer);
    input_manager_free(g->input_manager);
    renderer_destroy(&g->renderer);
    #if ASCII_MODE
        ascii_renderer_destroy(&g->ascii_renderer);
        ipc_destroy();
    #endif
    window_free(g->window);
    window_terminate();
    return 0;
}