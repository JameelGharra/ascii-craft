#ifndef _ascii_renderer_h_
#define _ascii_renderer_h_

#include <stdint.h>

typedef struct {
    // Terminal dimensions
    int ascii_width;
    int ascii_height;
    // OpenGL dimensions
    int source_width;
    int source_height;
} AsciiConfig;

typedef struct {
    uint8_t char_code;
    uint8_t r;
    uint8_t g;
    uint8_t b;
} AsciiPixel;

typedef struct AsciiRenderer AsciiRenderer;

AsciiRenderer *ascii_renderer_create(const AsciiConfig *config);
void ascii_renderer_destroy(AsciiRenderer **renderer_ptr);

void ascii_renderer_get_target_size(AsciiRenderer *renderer, int *width, int *height);
void ascii_renderer_bind_offscreen_buffer(AsciiRenderer *renderer);
void ascii_renderer_read_pixels(AsciiRenderer *renderer);
uint32_t ascii_renderer_render(AsciiRenderer *renderer, AsciiPixel *out);
#if ASCII_LOCAL_PRINT
    void ascii_renderer_print_debug(AsciiRenderer *renderer, const AsciiPixel *buffer);
#endif

// Retrieves performance stats.
void ascii_renderer_get_stats(
    AsciiRenderer *renderer,
    double *conversion_time_ms,
    double *fps
);

#endif