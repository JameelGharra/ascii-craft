#include "player.h"
#include "util.h"
#include "renderer.h"
#include <math.h>

// INTERNAL HELPERS //
static void _get_motion_vector(int flying, int sz, int sx, float rx, float ry,
                       float *vx, float *vy, float *vz);
// ========

void player_init(Player *player) {
    if(!player) {
        return ;
    }
    player->id = 0;
    player->name[0] = '\0';
    player->render_id = 0;
    player->flying = false;
    player->vertical_velocity = 0.0f; // was referred to as dy in original code
    player->render_id = INVALID_RENDERABLE_OBJECT_ID;
}
void player_update_look(Player *player, double m_dx, double m_dy, float m_sensitivity) {
    if(!player) {
        return ;
    }
    player->state.rx += m_dx * m_sensitivity;
    if (INVERT_MOUSE) {
        player->state.ry += m_dy * m_sensitivity;
    } 
    else {
        player->state.ry -= m_dy * m_sensitivity;
    }
    if (player->state.rx < 0)
    {
        player->state.rx += RADIANS(360);
    }
    if (player->state.rx >= RADIANS(360))
    {
        player->state.rx -= RADIANS(360);
    }
    player->state.ry = MAX(player->state.ry, -RADIANS(90));
    player->state.ry = MIN(player->state.ry, RADIANS(90));
}
void player_update_pos(Player *player, PlayerMovementIntent *intent, WorldQuery *world, double dt) {
    if (!player || !intent || !world) {
        return;
    }
    State *s = &player->state;
    int sx = 0, sz = 0;
    float m = dt * 1.0f;
    if (intent->forward)
        sz--;
    if (intent->backward)
        sz++;
    if (intent->left)
        sx--;
    if (intent->right)
        sx++;
    if (intent->turn_left)
        s->rx -= m;
    if (intent->turn_right)
        s->rx += m;
    if (intent->turn_up)
        s->ry += m;
    if (intent->turn_down)
        s->ry -= m;

    float vx, vy, vz;
    _get_motion_vector(player->flying, sz, sx, s->rx, s->ry, &vx, &vy, &vz);
    if(intent->jump) {
        if(player->flying) {
            vy = 1.0f;
        }
        else if(player->vertical_velocity == 0.0f) {
            player->vertical_velocity = 8.0f;
        }
    }
    float speed = player->flying ? 20.0f : 5.0f;
    PlayerCollisionInput pci = {
        .speed = speed,
        .vx = vx, .vy = vy, .vz = vz,
    };
    world_resolve_player_collision(world, player, &pci, dt);
}

// INTERNAL HELPERS IMPLEMENTATIONS //
static void _get_motion_vector(int flying, int sz, int sx, float rx, float ry,
                       float *vx, float *vy, float *vz)
{
    *vx = 0;
    *vy = 0;
    *vz = 0;
    if (!sz && !sx)
    {
        return;
    }
    float strafe = atan2f(sz, sx);
    if (flying)
    {
        float m = cosf(ry);
        float y = sinf(ry);
        if (sx)
        {
            if (!sz)
            {
                y = 0;
            }
            m = 1;
        }
        if (sz > 0)
        {
            y = -y;
        }
        *vx = cosf(rx + strafe) * m;
        *vy = y;
        *vz = sinf(rx + strafe) * m;
    }
    else
    {
        *vx = cosf(rx + strafe);
        *vy = 0;
        *vz = sinf(rx + strafe);
    }
}
