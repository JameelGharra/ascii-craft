#include "window.h"
#include <glfw3.h>
#include "config.h"
#include <glew.h>
#include "input_defs.h"


struct Window {
    GLFWwindow *handle;
    int width;
    int height;
    int scale;
    on_key_window_callback key_callback;
    on_char_window_callback char_callback;
    on_mouse_button_window_callback mouse_button_callback;
    on_scroll_window_callback scroll_callback;
};
int window_init() {
    if (!glfwInit()) {
        return 0;
    }
    return 1;
}
static void setup_glfw_callbacks(GLFWwindow *handle) {
    glfwSetKeyCallback(handle, glfw_key_callback_wrapper);
    glfwSetMouseButtonCallback(handle, glfw_mouse_button_callback_wrapper);
    glfwSetScrollCallback(handle, glfw_scroll_callback_wrapper);
    glfwSetCharCallback(handle, glfw_char_callback_wrapper);
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
    setup_glfw_callbacks(window->handle);
    return window;
}
void window_free(Window *window) {
    if (window) {
        if (window->handle) {
            glfwDestroyWindow(window->handle);
        }
        free(window);
    }
}
void window_terminate() {
    glfwTerminate();
}

int get_window_scale_factor(Window *window) {
    int window_width, window_height;
    int buffer_width, buffer_height;
    glfwGetWindowSize(window->handle, &window_width, &window_height);
    glfwGetFramebufferSize(window->handle, &buffer_width, &buffer_height);
    int result = buffer_width / window_width;
    result = MAX(1, result);
    result = MIN(2, result);
    return result;
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
// callbacks
void window_set_key_callback(Window *window, on_key_window_callback callback) {
    if (window && window->handle) {
        window->key_callback = callback;
    }
}
void window_set_char_callback(Window *window, on_char_window_callback callback) {
    if (window && window->handle) {
        window->char_callback = callback;
    }
}

void window_set_mouse_button_callback(Window *window, on_mouse_button_window_callback callback) {
    if (window && window->handle) {
        window->mouse_button_callback = callback;
    }
}
void window_set_scroll_callback(Window *window, on_scroll_window_callback callback) {
    if (window && window->handle) {
        window->scroll_callback = callback;
    }
}
// wrappers for callbacks
static void glfw_char_callback_wrapper(GLFWwindow* handle, unsigned int codepoint) {
    Window *window = (Window*)glfwGetWindowUserPointer(handle);
    if (window && window->char_callback) {
        window->char_callback(window, codepoint);
    }
}
static void glfw_key_callback_wrapper(GLFWwindow* handle, int key, int scancode, int action, int mods) {
    Window *window = (Window*)glfwGetWindowUserPointer(handle);
    if (window && window->key_callback) {
        InputKey input_key = translate_key_to_input(key);
        InputAction input_action = translate_action_to_input(action);
        InputMod input_mods = translate_mods_to_input(mods);
        if(input_key != KEY_INVALID && input_action != ACTION_INVALID) {
            window->key_callback(window, input_key, scancode, input_action, input_mods);
        }
    }
}
static void glfw_mouse_button_callback_wrapper(GLFWwindow* handle, int button, int action, int mods) {
    Window *window = (Window*)glfwGetWindowUserPointer(handle);
    if (window && window->mouse_button_callback) {
        InputMouseButton input_button = translate_mouse_button_to_input(button);
        InputAction input_action = translate_action_to_input(action);
        InputMod input_mods = translate_mods_to_input(mods);
        if(input_button != MOUSE_BUTTON_INVALID && input_action != ACTION_INVALID) {
            window->mouse_button_callback(window, input_button, input_action, input_mods);
        }
    }
}
static void glfw_scroll_callback_wrapper(GLFWwindow* handle, double xoffset, double yoffset) {
    Window *window = (Window*)glfwGetWindowUserPointer(handle);
    if (window && window->scroll_callback) {
        window->scroll_callback(window, xoffset, yoffset);
    }
}
// translation layer
static InputKey translate_key_to_input(int glfw_key) {
    switch (glfw_key) {
        case GLFW_KEY_BACKSPACE: return KEY_BACKSPACE;
        case GLFW_KEY_ESCAPE: return KEY_ESCAPE;
        case GLFW_KEY_ENTER: return KEY_ENTER;
        case GLFW_KEY_LEFT: return KEY_LEFT;
        case GLFW_KEY_RIGHT: return KEY_RIGHT;
        case GLFW_KEY_UP: return KEY_UP;
        case GLFW_KEY_DOWN: return KEY_DOWN;
        default: return KEY_INVALID;
    }
}
static InputAction translate_action_to_input(int glfw_action) {
    switch (glfw_action) {
        case GLFW_PRESS: return ACTION_PRESS;
        case GLFW_RELEASE: return ACTION_RELEASE;
        default: return ACTION_INVALID;
    }
}
static InputMouseButton translate_mouse_button_to_input(int glfw_button) {
    switch (glfw_button) {
        case GLFW_MOUSE_BUTTON_LEFT: return MOUSE_BUTTON_LEFT;
        case GLFW_MOUSE_BUTTON_RIGHT: return MOUSE_BUTTON_RIGHT;
        case GLFW_MOUSE_BUTTON_MIDDLE: return MOUSE_BUTTON_MIDDLE;
        default: return MOUSE_BUTTON_INVALID;
    }
}
static InputMod translate_mods_to_input(int glfw_mods) {
    InputMod input_mods = 0;
    if (glfw_mods & GLFW_MOD_SHIFT) {
        input_mods |= MOD_SHIFT;
    }
    if (glfw_mods & GLFW_MOD_CONTROL) {
        input_mods |= MOD_CONTROL;
    }
    if (glfw_mods & GLFW_MOD_SUPER) {
        input_mods |= MOD_SUPER;
    }
    return input_mods;
}