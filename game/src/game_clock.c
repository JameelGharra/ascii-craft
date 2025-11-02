#include <math.h>
#include "game_clock.h"
#include "time.h"

static struct {
    int day_length;
    double time_offset; // replaced glfwSetTIme internally
} clock_state;

void game_clock_init(int day_length_in_seconds) {
    clock_state.day_length = day_length_in_seconds;
    clock_state.time_offset = 0.0;
}
void game_clock_set_day_length(int day_length_in_seconds) {
    clock_state.day_length = day_length_in_seconds;
}

void game_clock_set_to(GameTimePresent preset) {
    float target_time_of_day = 0.0f;
    switch(preset) {
        case TIME_OF_DAY_MIDNIGHT:
            target_time_of_day = 0.0f;
            break;
        case TIME_OF_DAY_DAWN:
            target_time_of_day = 0.25f;
            break;
        case TIME_OF_DAY_MORNING:
            target_time_of_day = 1.0/3.0f;
            break;
        case TIME_OF_DAY_NOON:
            target_time_of_day = 0.5f;
            break;
        case TIME_OF_DAY_DUSK:
            target_time_of_day = 0.75f;
            break;
        case TIME_OF_DAY_NIGHT:
            target_time_of_day = 0.85f;
            break;
    }
    double target_raw_time = target_time_of_day * clock_state.day_length;
    double current_raw_time = time_get_seconds();
    clock_state.time_offset = target_raw_time - current_raw_time;
}
float game_clock_get_time_of_day() {
    if (clock_state.day_length <= 0){
        return 0.5; // mid-day
    }
    double now = time_get_seconds() + clock_state.time_offset;
    float t = now / clock_state.day_length;
    return fmodf(t, 1.0f);
}
float game_clock_get_daylight() {
    float timer = game_clock_get_time_of_day();
    if (timer < 0.5)
    {
        float t = (timer - 0.25) * 100;
        return 1 / (1 + powf(2, -t));
    }
    else
    {
        float t = (timer - 0.85) * 100;
        return 1 - 1 / (1 + powf(2, -t));
    }
}