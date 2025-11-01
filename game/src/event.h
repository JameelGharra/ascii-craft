#ifndef _event_h_
#define _event_h_

#include "input_defs.h"

typedef enum {
    EVENT_TYPE_KEY,
    EVENT_TYPE_MOUSE_BUTTON,
    EVENT_TYPE_SCROLL,
    EVENT_TYPE_CURSOR_POS,
} EventType;

typedef struct {
    EventType type;
    union {
        struct {
            InputKey key;
            int scancode;
            InputAction action;
            int mods;
        } key;
        struct {
            InputMouseButton button;
            InputAction action;
            int mods;
        } mouse_button;
        struct {
            double xoffset;
            double yoffset;
        } scroll;
        struct {
            double xpos;
            double ypos;
        } cursor_pos;
    } data;

} WindowEvent;


#endif