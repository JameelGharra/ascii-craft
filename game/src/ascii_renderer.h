#ifndef _ascii_renderer_h_
#define _ascii_renderer_h_


typedef struct {
    // Terminal dimensions
    int ascii_width;
    int ascii_height;
    // OpenGL dimensions
    int source_width;
    int source_height;
} AsciiConfig;

typedef struct AsciiRenderer AsciiRenderer;

AsciiRenderer *ascii_renderer_create(const AsciiConfig *config);
void ascii_renderer_destroy(AsciiRenderer **renderer_ptr);

void ascii_renderer_bind_offscreen_buffer(AsciiRenderer *renderer);
void ascii_renderer_read_pixels(AsciiRenderer *renderer);
void ascii_renderer_render_to_terminal(AsciiRenderer *renderer);

// Retrieves performance stats.
void ascii_renderer_get_stats(
    AsciiRenderer *renderer,
    double *conversion_time_ms,
    double *fps
);

#endif