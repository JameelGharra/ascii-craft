#include <stdio.h>
#include <stdbool.h>
#include <GLFW/glfw3.h>
#include "time.h"

static struct {
    bool is_initialized;
} time_state;

void time_init() {
    if(time_state.is_initialized) {
        return ;
    }
    time_state.is_initialized = true;
    // I did not add here any real init since we using GLFW and it was initialized prior for the window
}
void time_shutdown() {
    if(!time_state.is_initialized) {
        return ;
    }
    time_state.is_initialized = false;
    // No real shutdown needed for GLFW time
}
double time_get_seconds() {
    if(!time_state.is_initialized) {
        fprintf(stderr, "Warning: time_get_seconds() called before time_init()\n");
        return 0.0;
    }
    return glfwGetTime();
}
