#include <stdlib.h>
#include <GL/glew.h>
#include <GLFW/glfw3.h>
#include <stdint.h>
#include "window.h"
#include "input_defs.h"
#include "config.h"
#include "queue.h"
#include "util.h"

struct Window {
    GLFWwindow *handle;
    int width;
    int height;
    int scale;
    Queue *event_queue;
    double previous_mouse_x;
    double previous_mouse_y;
};

// INTERNAL HELPERS //

// wrappers for callbacks
static void _glfw_key_callback_wrapper(GLFWwindow* handle, int key, int scancode, int action, int mods);
static void _glfw_mouse_button_callback_wrapper(GLFWwindow* handle, int button, int action, int mods);
static void _glfw_scroll_callback_wrapper(GLFWwindow* handle, double xoffset, double yoffset);
static void _setup_glfw_callbacks(GLFWwindow *handle);
// translation layer
static InputKey _translate_key_to_input(int glfw_key);
static int _translate_input_to_glfw_key(InputKey key);
static InputAction _translate_action_to_input(int glfw_action);
static InputMouseButton _translate_mouse_button_to_input(int glfw_button);
static int _translate_mods_to_input(int glfw_mods);

// ========

int window_global_init() {
    if (!glfwInit()) {
        return 0;
    }
    return 1;
}
Window *window_create(int width, int height, const char* title, int fullscreen) {
    int window_width = width;
    int window_height = height;
    GLFWmonitor *monitor = NULL;
    if (fullscreen) {
        int mode_count;
        monitor = glfwGetPrimaryMonitor();
        const GLFWvidmode *modes = glfwGetVideoModes(monitor, &mode_count);
        window_width = modes[mode_count - 1].width;
        window_height = modes[mode_count - 1].height;
    }
    Window *window = (Window*)malloc(sizeof(Window));
    if(!window) {
        return NULL;
    }
    window->handle = glfwCreateWindow(
        window_width, window_height, title, monitor, NULL);
    if(window->handle == NULL) {
        free(window);
        return NULL;
    }
    glfwMakeContextCurrent(window->handle);
    if(glewInit() != GLEW_OK) {
        window_free(window);
        return NULL;
    }
    glfwSwapInterval(VSYNC);
    glfwSetInputMode(window->handle, GLFW_CURSOR, GLFW_CURSOR_DISABLED);
    glfwSetWindowUserPointer(window->handle, window);
    _setup_glfw_callbacks(window->handle);
    window->width = window_width;
    window->height = window_height;
    window->event_queue = queue_create();
    window_get_cursor_pos(window, &window->previous_mouse_x, &window->previous_mouse_y);
    window->previous_mouse_x = 0.0f;
    window->previous_mouse_y = 0.0f;
    if(!window->event_queue) {
        window_free(window);
        return NULL;
    }
    return window;
}
void window_free(Window *window) {
    if (window) {
        if (window->handle) {
            glfwDestroyWindow(window->handle);
        }
        if (window->event_queue) {
            while(!queue_is_empty(window->event_queue)) {
                WindowEvent *e = (WindowEvent*)queue_dequeue(window->event_queue);
                if(e) {
                    free(e);
                }
            }
            queue_destroy(window->event_queue);
        }
        free(window);
    }
}
void window_terminate() {
    glfwTerminate();
}

int window_get_scale_factor(Window *window) {
    if(!window || !window->handle) {
        return 1;
    }
    int window_width, window_height;
    int buffer_width, buffer_height;
    glfwGetWindowSize(window->handle, &window_width, &window_height);
    glfwGetFramebufferSize(window->handle, &buffer_width, &buffer_height);
    int result = buffer_width / window_width;
    result = MAX(1, result);
    result = MIN(2, result);
    window->scale = result;
    return result;
}
void window_get_size(Window *window, int *width, int *height) {
    if (window && width && height) {
        *width = window->width;
        *height = window->height;
    }
}
void window_update_size(Window *window) {
    if (window && window->handle){
        glfwGetFramebufferSize(window->handle, &window->width, &window->height);
    }
}
// inputs
void window_set_cursor_mode(Window *window, InputCursorMode mode) {
    if (window && window->handle) {
        int glfw_mode = GLFW_CURSOR_NORMAL;
        if (mode == CURSOR_DISABLED) {
            glfw_mode = GLFW_CURSOR_DISABLED;
        }
        glfwSetInputMode(window->handle, GLFW_CURSOR, glfw_mode);
    }
}
InputCursorMode window_get_cursor_mode(Window *window) {
    if (window && window->handle) {
        int mode = glfwGetInputMode(window->handle, GLFW_CURSOR);
        if (mode == GLFW_CURSOR_DISABLED) {
            return CURSOR_DISABLED;
        }
        return CURSOR_NORMAL;
    }
    return CURSOR_NORMAL;
}
void window_get_cursor_pos(Window *window, double *xpos, double *ypos) {
    if (window && window->handle && xpos && ypos) {
        glfwGetCursorPos(window->handle, xpos, ypos);
    }
}
bool window_next_frame(Window *window) {
    if (window && window->handle) {
        glfwSwapBuffers(window->handle);
        glfwPollEvents();
        if(glfwWindowShouldClose(window->handle)) {
            return false;
        }
        return true;
    }
    return false;
}
bool window_poll_event(Window *window, WindowEvent *event) {
    if (window && event) {
        if (!queue_is_empty(window->event_queue)) {
            WindowEvent *e = (WindowEvent*)queue_dequeue(window->event_queue);
            if (e) {
                *event = *e;
                free(e);
                return true;
            }
        }
    }
    return false;
}
bool window_is_key_down(Window *window, InputKey key) {
    if (window && window->handle) {
        int glfw_key = _translate_input_to_glfw_key(key);
        if (glfw_key != GLFW_KEY_UNKNOWN) {
            return glfwGetKey(window->handle, glfw_key);
        }
    }
    return false;
}
void window_get_cursor_delta(Window *window, double *dx, double *dy) {
    if(!window || !dx || !dy) {
        return ;
    }
    if(window_get_cursor_mode(window) == CURSOR_DISABLED && 
    (window->previous_mouse_x || window->previous_mouse_y)) {
        double current_x, current_y;
        window_get_cursor_pos(window, &current_x, &current_y);
        *dx = current_x - window->previous_mouse_x;
        *dy = current_y - window->previous_mouse_y;
        window->previous_mouse_x = current_x;
        window->previous_mouse_y = current_y;
    }
    else {
        window_get_cursor_pos(window, &window->previous_mouse_x, &window->previous_mouse_y);
    }
}

// INTERNAL HELPERS IMPLEMENTATIONS //

// wrappers for callbacks 
static void _glfw_key_callback_wrapper(GLFWwindow* handle, int key, int scancode, int action, int mods) {
    Window *window = (Window*)glfwGetWindowUserPointer(handle);
    if (window) {
        InputKey input_key = _translate_key_to_input(key);
        InputAction input_action = _translate_action_to_input(action);
        int input_mods = _translate_mods_to_input(mods);
        if(input_key != KEY_INVALID && input_action != ACTION_INVALID) {
            WindowEvent *event = (WindowEvent*)malloc(sizeof(WindowEvent));
            if(event) {
                event->type = EVENT_TYPE_KEY;
                event->data.key.key = input_key;
                event->data.key.scancode = scancode;
                event->data.key.action = input_action;
                event->data.key.mods = input_mods;
                queue_enqueue(window->event_queue, event);
            }
        }
    }
}
static void _glfw_mouse_button_callback_wrapper(GLFWwindow* handle, int button, int action, int mods) {
    Window *window = (Window*)glfwGetWindowUserPointer(handle);
    if (window) {
        InputMouseButton input_button = _translate_mouse_button_to_input(button);
        InputAction input_action = _translate_action_to_input(action);
        int input_mods = _translate_mods_to_input(mods);
        if(input_button != MOUSE_BUTTON_INVALID && input_action != ACTION_INVALID) {

            WindowEvent *event = (WindowEvent*)malloc(sizeof(WindowEvent));
            if(event) {
                event->type = EVENT_TYPE_MOUSE_BUTTON;
                event->data.mouse_button.button = input_button;
                event->data.mouse_button.action = input_action;
                event->data.mouse_button.mods = input_mods;
                queue_enqueue(window->event_queue, event);
            }
        }
    }
}
static void _glfw_scroll_callback_wrapper(GLFWwindow* handle, double xoffset, double yoffset) {
    Window *window = (Window*)glfwGetWindowUserPointer(handle);
    if (window) {
        WindowEvent *event = (WindowEvent*)malloc(sizeof(WindowEvent));
        if(event) {
            event->type = EVENT_TYPE_SCROLL;
            event->data.scroll.xoffset = xoffset;
            event->data.scroll.yoffset = yoffset;
            queue_enqueue(window->event_queue, event);
        }
    }
}
static void _setup_glfw_callbacks(GLFWwindow *handle) {
    glfwSetKeyCallback(handle, _glfw_key_callback_wrapper);
    glfwSetMouseButtonCallback(handle, _glfw_mouse_button_callback_wrapper);
    glfwSetScrollCallback(handle, _glfw_scroll_callback_wrapper);
}

// translation layer
static InputKey _translate_key_to_input(int glfw_key) {
    if(glfw_key >= GLFW_KEY_A && glfw_key <= GLFW_KEY_Z) {
        return (InputKey)(KEY_A + (glfw_key - GLFW_KEY_A));
    }
    switch (glfw_key) {
        case GLFW_KEY_BACKSPACE: return KEY_BACKSPACE;
        case GLFW_KEY_ESCAPE: return KEY_ESCAPE;
        case GLFW_KEY_ENTER: return KEY_ENTER;
        case GLFW_KEY_LEFT: return KEY_LEFT;
        case GLFW_KEY_RIGHT: return KEY_RIGHT;
        case GLFW_KEY_UP: return KEY_UP;
        case GLFW_KEY_DOWN: return KEY_DOWN;
        case GLFW_KEY_TAB: return KEY_TAB;
        case GLFW_KEY_SPACE: return KEY_SPACE;
        case GLFW_KEY_LEFT_SHIFT: return KEY_LEFT_SHIFT;
        case GLFW_KEY_0: case GLFW_KEY_1: case GLFW_KEY_2: case GLFW_KEY_3: case GLFW_KEY_4:
             case GLFW_KEY_5: case GLFW_KEY_6: case GLFW_KEY_7: case GLFW_KEY_8: case GLFW_KEY_9:
            return (InputKey)(KEY_0 + (glfw_key - GLFW_KEY_0));
        default: return KEY_INVALID;
    }
}
static int _translate_input_to_glfw_key(InputKey key) {
    if(key >= KEY_A && key <= KEY_Z) {
        return GLFW_KEY_A + (key - KEY_A);
    }
    switch (key) {
        case KEY_BACKSPACE: return GLFW_KEY_BACKSPACE;
        case KEY_ESCAPE: return GLFW_KEY_ESCAPE;
        case KEY_ENTER: return GLFW_KEY_ENTER;
        case KEY_LEFT: return GLFW_KEY_LEFT;
        case KEY_RIGHT: return GLFW_KEY_RIGHT;
        case KEY_UP: return GLFW_KEY_UP;
        case KEY_DOWN: return GLFW_KEY_DOWN;
        case KEY_TAB: return GLFW_KEY_TAB;
        case KEY_SPACE: return GLFW_KEY_SPACE;
        case KEY_LEFT_SHIFT: return GLFW_KEY_LEFT_SHIFT;
        case KEY_0: case KEY_1: case KEY_2: case KEY_3: case KEY_4:
             case KEY_5: case KEY_6: case KEY_7: case KEY_8: case KEY_9:
            return GLFW_KEY_0 + (key - KEY_0);
        default: return GLFW_KEY_UNKNOWN;
    }
}
static InputAction _translate_action_to_input(int glfw_action) {
    switch (glfw_action) {
        case GLFW_PRESS: return ACTION_PRESS;
        case GLFW_RELEASE: return ACTION_RELEASE;
        default: return ACTION_INVALID;
    }
}
static InputMouseButton _translate_mouse_button_to_input(int glfw_button) {
    switch (glfw_button) {
        case GLFW_MOUSE_BUTTON_LEFT: return MOUSE_BUTTON_LEFT;
        case GLFW_MOUSE_BUTTON_RIGHT: return MOUSE_BUTTON_RIGHT;
        case GLFW_MOUSE_BUTTON_MIDDLE: return MOUSE_BUTTON_MIDDLE;
        default: return MOUSE_BUTTON_INVALID;
    }
}
static int _translate_mods_to_input(int glfw_mods) {
    int input_mods = 0;
    if (glfw_mods & GLFW_MOD_SHIFT) {
        input_mods |= INPUT_MOD_SHIFT;
    }
    if (glfw_mods & GLFW_MOD_CONTROL) {
        input_mods |= INPUT_MOD_CONTROL;
    }
    if (glfw_mods & GLFW_MOD_SUPER) {
        input_mods |= INPUT_MOD_SUPER;
    }
    return input_mods;
}