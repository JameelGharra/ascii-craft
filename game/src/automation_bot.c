#include "automation_bot.h"
#include <math.h>
#include "util.h"

void automation_bot_reset(AutomationBot *bot) {
    if (!bot) {
        return;
    }
    bot->active_move_dir = MOVE_DIR_NONE;
    bot->move_timer = 0.0f;
    bot->is_rotating = false;
    bot->holding_jump = false;
    bot->rotate_timer = 0.0f;
    bot->rotate_duration = 0.0f;
    bot->start_rx = 0.0f;
    bot->start_ry = 0.0f;
    bot->target_rx = 0.0f;
    bot->target_ry = 0.0f;
}

void automation_bot_pos_update(AutomationBot *bot, Player *player, PlayerMovementIntent *move_intent, double dt) {
    if(!bot || !player || !move_intent) {
        return;
    }
    if (bot->move_timer > 0) {
        bot->move_timer -= dt;
        switch (bot->active_move_dir) {
            case MOVE_DIR_FORWARD: move_intent->forward = true; break;
            case MOVE_DIR_BACKWARD: move_intent->backward = true; break;
            case MOVE_DIR_LEFT: move_intent->left = true; break;
            case MOVE_DIR_RIGHT: move_intent->right = true; break;
        }
        move_intent->jump = bot->holding_jump;

        if (bot->move_timer <= 0) {
            bot->active_move_dir = MOVE_DIR_NONE; // stop
            bot->holding_jump = false;
        }
    }
}
void automation_bot_look_update(AutomationBot *bot, Player *player, double dt) {
    if (!bot || !player) {
        return;
    }
    if (bot->is_rotating) {
        bot->rotate_timer += dt;
        float t = bot->rotate_timer / bot->rotate_duration;
        
        if (t >= 1.0f) {
            // finishing the rotation
            player->state.rx = bot->target_rx;
            player->state.ry = bot->target_ry;
            // normalizing yaw rx to [0, 2PI)
            // this prevents float precision degradation after thousands of turns
            while (player->state.rx < 0.0f) {
                player->state.rx += (float)(2.0 * PI);
            }
            while (player->state.rx >= (float)(2.0 * PI)) {
                player->state.rx -= (float)(2.0 * PI);
            }
            bot->is_rotating = false;
        } else {
            // interpolate (simple lerp)
            player->state.rx = bot->start_rx + (bot->target_rx - bot->start_rx) * t;
            player->state.ry = bot->start_ry + (bot->target_ry - bot->start_ry) * t;
        }
    }
}
