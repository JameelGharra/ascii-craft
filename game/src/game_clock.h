#ifndef _game_clock_h_
#define _game_clock_h_

typedef enum {
    TIME_OF_DAY_MIDNIGHT, // 0.0
    TIME_OF_DAY_DAWN,     // 0.25
    TIME_OF_DAY_MORNING,  // 0.33
    TIME_OF_DAY_NOON,     // 0.5
    TIME_OF_DAY_DUSK,     // 0.75
    TIME_OF_DAY_NIGHT     // 0.85
} GameTimePresent;

void game_clock_init(int day_length_in_seconds);
void game_clock_set_day_length(int day_length_in_seconds);

float game_clock_get_time_of_day(); // time of day in the game world (0.0 to 1.0)
float game_clock_get_daylight(); // daylight level based on the time
void game_clock_set_to(GameTimePresent preset);

#endif