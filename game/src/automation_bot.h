#ifndef _automation_bot_h_
#define _automation_bot_h_

#include <stdbool.h>
#include "player.h"

enum MoveDirection {
    MOVE_DIR_NONE = 0,
    MOVE_DIR_FORWARD,
    MOVE_DIR_BACKWARD,
    MOVE_DIR_LEFT,
    MOVE_DIR_RIGHT
};

typedef struct {
    int active_move_dir;
    float move_timer;

    bool is_rotating;
    float rotate_timer;
    float rotate_duration;
    float start_rx, start_ry;
    float target_rx, target_ry;

} AutomationBot;

void automation_bot_reset(AutomationBot *bot); // the caller owns the mem. due to public struct
void automation_bot_pos_update(AutomationBot *bot, Player *player, PlayerMovementIntent *move_intent, double dt);
void automation_bot_look_update(AutomationBot *bot, Player *player, double dt);
#endif