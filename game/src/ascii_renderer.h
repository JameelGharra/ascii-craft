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

// Creates and initializes the renderer and all its resources (FBO, buffers).
AsciiRenderer* ascii_renderer_create(const AsciiConfig* config);

// Frees all memory and OpenGL resources.
void ascii_renderer_destroy(AsciiRenderer** renderer_ptr);

// Binds the internal FBO so that subsequent OpenGL calls render to it.
void ascii_renderer_bind_offscreen_buffer(AsciiRenderer *renderer);

// Reads the rendered pixels from the FBO to a CPU-side buffer.
void ascii_renderer_read_pixels(AsciiRenderer *renderer);

// Converts the pixel buffer to an ASCII frame.
void ascii_renderer_render_to_terminal(AsciiRenderer *renderer);

// Retrieves performance stats.
void ascii_renderer_get_stats(
    AsciiRenderer *renderer,
    double *conversion_time_ms,
    double *fps
);

#endif