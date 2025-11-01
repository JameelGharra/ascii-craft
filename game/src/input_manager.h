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
    COMMAND_TOGGLE_ORTHO_MODE, // F key for 2D like orthographic projection
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
    } data;
} GameCommand;

InputManager *input_manager_create();
void input_manager_free(InputManager *manager);
void input_manager_update(InputManager *manager, Window *window);
bool input_manager_get_next_command(InputManager *manager, GameCommand *command);

#endif