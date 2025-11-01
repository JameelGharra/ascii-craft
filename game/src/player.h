#ifndef _player_h_
#define _player_h_

#include <stdbool.h>
#include "config.h"
#include "world_query.h"

typedef unsigned int RenderableObjectID;

typedef struct {
    float x;
    float y;
    float z;
    float rx;
    float ry;
    float t;
} State;

typedef struct {
    bool forward;
    bool backward;
    bool left;
    bool right;
    bool turn_left; // using arrows
    bool turn_right;
    bool turn_up;
    bool turn_down;
    bool jump;

} PlayerMovementIntent;

struct Player {
    int id;
    char name[MAX_NAME_LENGTH];
    State state;
    State state1;
    State state2;
    bool flying;
    float vertical_velocity;
    RenderableObjectID render_id;
};
typedef struct Player Player;

void player_init(Player *player);
void player_update_look(Player *player, double m_dx, double m_dy, float m_sensitivity);
void player_update_pos(Player *player, PlayerMovementIntent *intent, WorldQuery *world, double dt);

#endif