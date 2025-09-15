#ifndef _window_h_
#define _window_h_

typedef struct Window Window;

// callback function types
typedef void (*on_key_window_callback)(Window *window, InputKey key, int scancode, InputAction action, InputMod mods);
typedef void (*on_char_window_callback)(Window *window, unsigned int codepoint);
typedef void (*on_mouse_button_window_callback)(Window *window, InputMouseButton button, InputAction action, InputMod mods);
typedef void (*on_scroll_window_callback)(Window *window, double xoffset, double yoffset);


int window_init();
Window *window_create(int width, int height, const char* title, int fullscreen);
void window_free(Window *window);
void window_terminate();
int get_window_scale_factor(Window *window);

// callbacks
void window_set_key_callback(Window *window, on_key_window_callback callback);
void window_set_char_callback(Window *window, on_char_window_callback callback);
void window_set_mouse_button_callback(Window *window, on_mouse_button_window_callback callback);
void window_set_scroll_callback(Window *window, on_scroll_window_callback callback);

// inputs
int window_get_cursor_mode(Window *window);
void window_set_cursor_mode(Window *window, InputCursorMode mode);
void window_get_cursor_pos(Window *window, double *xpos, double *ypos);

#endif