#ifndef _input_defs_h_
#define _input_defs_h_

typedef enum {
    MOD_SHIFT = 0x0001,
    MOD_CONTROL = 0x0002,
    MOD_SUPER = 0x0008,

} InputMod;

typedef enum {
    ACTION_INVALID = -1,
    ACTION_RELEASE = 0,
    ACTION_PRESS = 1,
} InputAction;


typedef enum {
    CURSOR_NORMAL,
    CURSOR_DISABLED,
} InputCursorMode;

typedef enum {
    KEY_INVALID = -1,
    KEY_BACKSPACE = 1,
    KEY_ESCAPE = 2,
    KEY_ENTER = 3,
    KEY_LEFT = 4,
    KEY_RIGHT = 5,
    KEY_UP = 6,
    KEY_DOWN = 7,
} InputKey;

typedef enum {
    MOUSE_BUTTON_INVALID = -1,
    MOUSE_BUTTON_LEFT = 0,
    MOUSE_BUTTON_RIGHT = 1,
    MOUSE_BUTTON_MIDDLE = 2,
} InputMouseButton;


#endif