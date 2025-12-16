#include <stdlib.h>
#include <stdbool.h>
#include <stdio.h>
#include "input_manager.h"
#include "queue.h"
#include "config.h"
#include "ipc.h"

struct InputManager {
    Queue *command_queue;
};

// INTERNAL HELPERS //
static bool _construct_key_command(Window *window, WindowEvent *event, GameCommand *command);
static void _construct_scroll_command(WindowEvent *event, GameCommand *command);
static bool _construct_mouse_button_command(Window *window, WindowEvent *event, GameCommand *command);
#if AUTOMATION_BOT_MODE
    static void _process_ipc_commands(InputManager *manager);
#endif
// ========

void input_manager_update(InputManager *manager, Window *window) {
    if(!manager || !window) {
        return ;
    }
    #if AUTOMATION_BOT_MODE
        _process_ipc_commands(manager);
    #endif
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
#if AUTOMATION_BOT_MODE
static void _process_ipc_commands(InputManager *manager) {
    IPCCommandEntry ipc_cmd;
    while(ipc_read_command(&ipc_cmd)) {
        GameCommand *game_cmd = (GameCommand*)malloc(sizeof(GameCommand));
        if(!game_cmd) {
            continue; // for now on failure, it will loop forever and freeze main thread
        }
        bool valid = true;
        switch(ipc_cmd.type) {
            case IPC_CMD_BACKWARD: case IPC_CMD_FORWARD: case IPC_CMD_LEFT: case IPC_CMD_RIGHT:
                // im planning to simulate press duration since currently the input sys is continuous events-based
                game_cmd->type = COMMAND_MOVE_FORWARD+(ipc_cmd.type-IPC_CMD_FORWARD);
                game_cmd->data.movement.duration = BOT_MOVE_DURATION;
                game_cmd->data.movement.jump = false;
                break;
            case IPC_CMD_JUMP_FORWARD: case IPC_CMD_JUMP_BACKWARD:
            case IPC_CMD_JUMP_LEFT: case IPC_CMD_JUMP_RIGHT:
                game_cmd->type = COMMAND_MOVE_FORWARD+(ipc_cmd.type-IPC_CMD_JUMP_FORWARD);
                game_cmd->data.movement.duration = BOT_MOVE_DURATION;
                game_cmd->data.movement.jump = true;
                break;
            case IPC_CMD_TURN_LEFT: case IPC_CMD_TURN_RIGHT:
                game_cmd->type = COMMAND_LOOK_YAW;
                game_cmd->data.look.angle_delta = (ipc_cmd.type == IPC_CMD_TURN_LEFT) ? -BOT_TURN_ANGLE : BOT_TURN_ANGLE;
                game_cmd->data.look.duration = BOT_TURN_DURATION;           
                break;
            case IPC_CMD_LOOK_UP: case IPC_CMD_LOOK_DOWN:
                game_cmd->type = COMMAND_LOOK_PITCH;
                game_cmd->data.look.angle_delta = (ipc_cmd.type == IPC_CMD_LOOK_UP) ? BOT_LOOK_ANGLE : -BOT_LOOK_ANGLE;
                game_cmd->data.look.duration = BOT_TURN_DURATION;
                break;
            case IPC_CMD_FLY:
                game_cmd->type = COMMAND_TOGGLE_FLY;
                break;
            case IPC_CMD_BUILD:
                game_cmd->type = COMMAND_BUILD;
                break;
            case IPC_CMD_DESTROY:
                game_cmd->type = COMMAND_DESTROY;
                break;
            case IPC_CMD_SELECT_SLOT:
                game_cmd->type = COMMAND_SET_ITEM_INDEX;
                game_cmd->data.set_item.index = ipc_cmd.value;
                break;
            default:
                valid = false;
        }
        if(valid) {
            queue_enqueue(manager->command_queue, game_cmd);

        }
        else {
            free(game_cmd);
        }
    }
}
#endif