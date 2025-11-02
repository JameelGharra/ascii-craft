#ifndef _input_defs_h_
#define _input_defs_h_

#define INPUT_MOD_SHIFT 0x0001
#define INPUT_MOD_CONTROL 0x0002
#define INPUT_MOD_SUPER 0x0008

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
    KEY_TAB = 8,
    KEY_SPACE = 9,
    KEY_LEFT_SHIFT = 10,
    KEY_0 = '0',
    KEY_1,
    KEY_2,
    KEY_3,
    KEY_4,
    KEY_5,
    KEY_6,
    KEY_7,
    KEY_8,
    KEY_9,
    KEY_A = 'A', 
    KEY_B, 
    KEY_C, 
    KEY_D, 
    KEY_E, 
    KEY_F, 
    KEY_G, 
    KEY_H, 
    KEY_I, 
    KEY_J,
    KEY_K, 
    KEY_L, 
    KEY_M, 
    KEY_N, 
    KEY_O, 
    KEY_P, 
    KEY_Q, 
    KEY_R, 
    KEY_S, 
    KEY_T, 
    KEY_U, 
    KEY_V, 
    KEY_W, 
    KEY_X, 
    KEY_Y, 
    KEY_Z,
} InputKey;

typedef enum {
    MOUSE_BUTTON_INVALID = -1,
    MOUSE_BUTTON_LEFT = 0,
    MOUSE_BUTTON_RIGHT = 1,
    MOUSE_BUTTON_MIDDLE = 2,
} InputMouseButton;


#endif