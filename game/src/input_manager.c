#include <stdlib.h>
#include <stdbool.h>
#include "input_manager.h"
#include "queue.h"
#include "config.h"

struct InputManager {
    Queue *command_queue;
};

// INTERNAL HELPERS //
static bool _construct_key_command(Window *window, WindowEvent *event, GameCommand *command);
static void _construct_scroll_command(WindowEvent *event, GameCommand *command);
static bool _construct_mouse_button_command(Window *window, WindowEvent *event, GameCommand *command);
// ========

void input_manager_update(InputManager *manager, Window *window) {
    if(!manager || !window) {
        return ;
    }
    WindowEvent event;
    while(window_poll_event(window, &event)) {
        GameCommand *command = (GameCommand *)malloc(sizeof(GameCommand));
        if(!command) {
            return ;
        }
        switch(event.type) {
            case EVENT_TYPE_KEY:
                if(!_construct_key_command(window, &event, command)) {
                    free(command);
                    continue;
                }
                break;
            case EVENT_TYPE_SCROLL:
                _construct_scroll_command(&event, command);
                break;
            
            case EVENT_TYPE_MOUSE_BUTTON:
                if(!_construct_mouse_button_command(window, &event, command)) {
                    free(command);
                    continue;
                }
                break;
            default:
                free(command);
                continue;
        }
        queue_enqueue(manager->command_queue, command);
    }
}
bool input_manager_get_next_command(InputManager *manager, GameCommand *command) {
    if(!manager || !command) {
        return false;
    }
    GameCommand *next_command = (GameCommand *)queue_dequeue(manager->command_queue);
    if(!next_command) {
        return false;
    }
    *command = *next_command;
    free(next_command);
    return true;
}
InputManager *input_manager_create() {
    InputManager *manager = (InputManager *)malloc(sizeof(InputManager));
    if(!manager) {
        return NULL;
    }
    manager->command_queue = queue_create();
    if(!manager->command_queue) {
        free(manager);
        return NULL;
    }
    return manager;
}
void input_manager_free(InputManager *manager) {
    if(manager) {
        if(manager->command_queue) {
            while(!queue_is_empty(manager->command_queue)) {
                GameCommand *cmd = (GameCommand *)queue_dequeue(manager->command_queue);
                if(cmd) {
                    free(cmd);
                }
            }
            queue_destroy(manager->command_queue);
        }
        free(manager);
    }
}
// INTERNAL HELPERS IMPLEMENTATIONS //
static bool _construct_key_command(Window *window, WindowEvent *event, GameCommand *command) {
    if(event->data.key.action != ACTION_PRESS) {
        return false;
    }
    int control = event->data.key.mods & (INPUT_MOD_CONTROL | INPUT_MOD_SUPER);
    int exclusive = window_get_cursor_mode(window) == CURSOR_DISABLED;
    if(event->data.key.key == KEY_ESCAPE && exclusive) {
        window_set_cursor_mode(window, CURSOR_NORMAL);
        return false;
    }
    if(event->data.key.key == KEY_ENTER) {
        if(control) {
            command->type = COMMAND_BUILD;
        } else {
            command->type = COMMAND_DESTROY;
        }
        return true;
    }
    if(event->data.key.key == KEY_TAB) {
        command->type = COMMAND_TOGGLE_FLY;
        return true;
    }
    if((event->data.key.key >= '1' && event->data.key.key <= '9') || event->data.key.key == '0') {
        command->type = COMMAND_SET_ITEM_INDEX;
        if(event->data.key.key == '0') {
            command->data.set_item.index = 9;
        } else {
            command->data.set_item.index = event->data.key.key - '1';
        }
        return true;
    }
    if(event->data.key.key == CRAFT_KEY_ITEM_NEXT) {
        command->type = COMMAND_CYCLE_UP_ITEM;
        return true;
    }
    if(event->data.key.key == CRAFT_KEY_ITEM_PREV) {
        command->type = COMMAND_CYCLE_DOWN_ITEM;
        return true;
    }
    return false;
}
static void _construct_scroll_command(WindowEvent *event, GameCommand *command) {
    command->type = COMMAND_SCROLL;
    command->data.cycle_item.y_delta = event->data.scroll.yoffset;
}
static bool _construct_mouse_button_command(Window *window, WindowEvent *event, GameCommand *command) {
    int control = event->data.mouse_button.mods & (INPUT_MOD_CONTROL | INPUT_MOD_SUPER);
    int exclusive = window_get_cursor_mode(window) == CURSOR_DISABLED;
    if(event->data.mouse_button.action != ACTION_PRESS) {
        return false;
    }
    if(event->data.mouse_button.button == MOUSE_BUTTON_LEFT) {
        if(exclusive) {
            if(control) {
                command->type = COMMAND_BUILD;
            } else {
                command->type = COMMAND_DESTROY;
            }
            return true;
        } else {
            window_set_cursor_mode(window, CURSOR_DISABLED);
            return false;
        }
    }
    if(event->data.mouse_button.button == MOUSE_BUTTON_RIGHT) {
        if(exclusive) {
            if(control) {
                command->type = COMMAND_TOGGLE_LIGHT;
            } else {
                command->type = COMMAND_BUILD;
            }
            return true;
        }
    }
    if(event->data.mouse_button.button == MOUSE_BUTTON_MIDDLE) {
        if(exclusive) {
            command->type = COMMAND_MIDDLE_COPY_ELEMENT;
            return true;
        }
    }
    return false;
}