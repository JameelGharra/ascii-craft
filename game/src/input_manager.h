#ifndef _input_h_
#define _input_h_

#include <stdbool.h>
#include "window.h"

typedef struct InputManager InputManager;

typedef enum {
    COMMAND_TOGGLE_FLY,
    COMMAND_BUILD, // right click
    COMMAND_DESTROY, // left click
    COMMAND_SET_ITEM_INDEX,
    COMMAND_CYCLE_UP_ITEM,
    COMMAND_CYCLE_DOWN_ITEM,
    COMMAND_SCROLL,
    COMMAND_MIDDLE_COPY_ELEMENT, // middle click
    COMMAND_TOGGLE_LIGHT,
    
    // duration based movement
    COMMAND_MOVE_FORWARD,
    COMMAND_MOVE_BACKWARD,
    COMMAND_MOVE_LEFT,
    COMMAND_MOVE_RIGHT,

    // angle + duration based camera movement
    COMMAND_LOOK_YAW, // for left/right
    COMMAND_LOOK_PITCH, // for up/down
    
    // kinda useless since we have casual jumps, but might be helpful in future features
    COMMAND_JUMP, 

} GameCommandType;

typedef struct {
    GameCommandType type;
    union {
        struct {
            int index;
        } set_item;
        struct {
            double y_delta; // y offset for scrolling
        } cycle_item;
        struct {
            float duration; // mimicking holding key in seconds
            bool jump;
        } movement;
        struct {
            float angle_delta;
            float duration; // how fast to turn (smoothness)
        } look;
    } data;
} GameCommand;

InputManager *input_manager_create();
void input_manager_free(InputManager *manager);
void input_manager_update(InputManager *manager, Window *window);
bool input_manager_get_next_command(InputManager *manager, GameCommand *command);

#endif