#ifndef _window_h_
#define _window_h_

#include "input_defs.h"
#include "event.h"
#include <stdbool.h>

typedef struct Window Window;


int window_global_init();
Window *window_create(int width, int height, const char* title, int fullscreen);
void window_free(Window *window);
void window_terminate(); // global termination (for all windows)
int window_get_scale_factor(Window *window);
void window_update_size(Window *window);
void window_get_size(Window *window, int *width, int *height);

// inputs
InputCursorMode window_get_cursor_mode(Window *window);
void window_set_cursor_mode(Window *window, InputCursorMode mode);
void window_get_cursor_pos(Window *window, double *xpos, double *ypos);
bool window_next_frame(Window *window); // swaps, polls &fills event queue, to be called once per frame
bool window_poll_event(Window *window, WindowEvent *event);

// for continuous events
bool window_is_key_down(Window *window, InputKey key);
void window_get_cursor_delta(Window *window, double *dx, double *dy);

#endif