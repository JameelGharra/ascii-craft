#ifndef _config_h_
#define _config_h_

#include "input_defs.h"

// app parameters
#define DEBUG 0
#define FULLSCREEN 0
#define WINDOW_WIDTH 1024
#define WINDOW_HEIGHT 768
#define WINDOW_NAME "Craft"
#define VSYNC 0
#define SCROLL_THRESHOLD 0.1
#define MAX_MESSAGES 4
#define DB_PATH "craft.db"
#define USE_CACHE 1
#define DAY_LENGTH 30 // was 600 initially
#define INVERT_MOUSE 0
#define MOUSE_SENSITIVITY 0.0025f

// rendering options
#define SHOW_LIGHTS 1
#define SHOW_PLANTS 1
#define SHOW_CLOUDS 1
#define SHOW_TREES 1
#define SHOW_ITEM 1
#define SHOW_CROSSHAIRS 1
#define SHOW_WIREFRAME 1
#define SHOW_INFO_TEXT 1
#define SHOW_CHAT_TEXT 1
#define SHOW_PLAYER_NAMES 1

// key bindings
#define CRAFT_KEY_FORWARD KEY_W
#define CRAFT_KEY_BACKWARD KEY_S
#define CRAFT_KEY_LEFT KEY_A
#define CRAFT_KEY_RIGHT KEY_D
#define CRAFT_KEY_JUMP KEY_SPACE
#define CRAFT_KEY_FLY KEY_TAB
#define CRAFT_KEY_OBSERVE KEY_O
#define CRAFT_KEY_OBSERVE_INSET KEY_P
#define CRAFT_KEY_ITEM_NEXT KEY_E
#define CRAFT_KEY_ITEM_PREV KEY_R
#define CRAFT_KEY_ZOOM KEY_LEFT_SHIFT
#define CRAFT_KEY_ORTHO KEY_F
#define CRAFT_KEY_CHAT KEY_T
#define CRAFT_KEY_COMMAND '/'
#define CRAFT_KEY_SIGN '`'

// advanced parameters
#define CREATE_CHUNK_RADIUS 10
#define RENDER_CHUNK_RADIUS 10
#define RENDER_SIGN_RADIUS 4
#define DELETE_CHUNK_RADIUS 14
#define CHUNK_SIZE 32
#define COMMIT_INTERVAL 5

// Maxs
#define MAX_PLAYERS 128
#define MAX_NAME_LENGTH 32
#define MAX_CHUNKS 8192

#endif
