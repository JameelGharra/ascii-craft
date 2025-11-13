#include "ascii_renderer.h"
#include <GL/glew.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include "time.h"

static const char* ASCII_PALETTE = " .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$";
static const int PALETTE_COUNT = 70; // did not include \0 in this count

struct AsciiRenderer {
    AsciiConfig config;

    // FBO resources for off-screen rendering
    GLuint fbo_handle;
    GLuint texture_handle;
    GLuint depth_buffer_handle;

    // CPU-side buffers (allocated once, reused every frame)
    unsigned char* pixel_buffer; // the high-res pixels read from the GPU
    char* frame_buffer;          // the final low-res ASCII string
    size_t frame_buffer_size;

    // For performance stats
    double conversion_time_ms;
    double frame_start_time;
    double fps;
};

AsciiRenderer* ascii_renderer_create(const AsciiConfig* config) {
    AsciiRenderer* renderer = (AsciiRenderer*)malloc(sizeof(AsciiRenderer));
    if (!renderer) return NULL;

    renderer->config = *config;

    size_t pixel_buffer_size = config->source_width * config->source_height * 3; // RGB
    renderer->pixel_buffer = (unsigned char*)malloc(pixel_buffer_size);
    
    // Generous estimate for the frame buffer size
    // (Max ANSI color code len + 1 char) * num_chars + num_newlines + reset code + null terminator
    renderer->frame_buffer_size = (19 + 1) * config->ascii_width * config->ascii_height + config->ascii_height + 5;
    renderer->frame_buffer = (char*)malloc(renderer->frame_buffer_size);

    glGenFramebuffers(1, &renderer->fbo_handle);
    glBindFramebuffer(GL_FRAMEBUFFER, renderer->fbo_handle);

    glGenTextures(1, &renderer->texture_handle);
    glBindTexture(GL_TEXTURE_2D, renderer->texture_handle);
    // Last param NULL allocates empty texture and not upload anything from CPU
    glTexImage2D(GL_TEXTURE_2D, 0, GL_RGB, config->source_width, config->source_height, 0, GL_RGB, GL_UNSIGNED_BYTE, NULL);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
    glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
    glFramebufferTexture2D(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, renderer->texture_handle, 0);

    glGenRenderbuffers(1, &renderer->depth_buffer_handle);
    glBindRenderbuffer(GL_RENDERBUFFER, renderer->depth_buffer_handle);
    glRenderbufferStorage(GL_RENDERBUFFER, GL_DEPTH24_STENCIL8, config->source_width, config->source_height);
    glFramebufferRenderbuffer(GL_FRAMEBUFFER, GL_DEPTH_STENCIL_ATTACHMENT, GL_RENDERBUFFER, renderer->depth_buffer_handle);

    if (glCheckFramebufferStatus(GL_FRAMEBUFFER) != GL_FRAMEBUFFER_COMPLETE) {
        fprintf(stderr, "Error: Off-screen framebuffer is not complete!\n");
        free(renderer->pixel_buffer);
        free(renderer->frame_buffer);
        free(renderer);
        return NULL;
    }
    glBindFramebuffer(GL_FRAMEBUFFER, 0);

    renderer->conversion_time_ms = 0.0;
    renderer->fps = 0.0;
    renderer->frame_start_time = 0.0;

    return renderer;
}

void ascii_renderer_destroy(AsciiRenderer** renderer_ptr) {
    if (!renderer_ptr || !*renderer_ptr) return;
    AsciiRenderer* renderer = *renderer_ptr;

    free(renderer->pixel_buffer);
    free(renderer->frame_buffer);

    glDeleteFramebuffers(1, &renderer->fbo_handle);
    glDeleteTextures(1, &renderer->texture_handle);
    glDeleteRenderbuffers(1, &renderer->depth_buffer_handle);

    free(renderer);
    *renderer_ptr = NULL;
}

void ascii_renderer_bind_offscreen_buffer(AsciiRenderer *renderer) {
    glBindFramebuffer(GL_FRAMEBUFFER, renderer->fbo_handle);
    glViewport(0, 0, renderer->config.source_width, renderer->config.source_height);
}

void ascii_renderer_read_pixels(AsciiRenderer *renderer) {
    // Set alignment to 1 to avoid issues with widths that aren't multiples of 4
    glPixelStorei(GL_PACK_ALIGNMENT, 1);
    glReadPixels(0, 0, renderer->config.source_width, renderer->config.source_height, GL_RGB, GL_UNSIGNED_BYTE, renderer->pixel_buffer);
    glBindFramebuffer(GL_FRAMEBUFFER, 0);
}

void ascii_renderer_render_to_terminal(AsciiRenderer *renderer) {

    double start_time = time_get_seconds();
    char* buf_ptr = renderer->frame_buffer;
    uint32_t last_color = 0xFFFFFFFF; // Impossible color to force first write

    // Scaling factors
    const float block_width = (float)renderer->config.source_width / renderer->config.ascii_width;
    const float block_height = (float)renderer->config.source_height / renderer->config.ascii_height;
    
    buf_ptr += sprintf(buf_ptr, "\033[;H"); // cursor to top-left

    for (int y = 0; y < renderer->config.ascii_height; ++y) {
        for (int x = 0; x < renderer->config.ascii_width; ++x) {
            
            // Block averaging
            int start_py = (int)(y * block_height);
            int end_py = (int)((y + 1) * block_height);
            int start_px = (int)(x * block_width);
            int end_px = (int)((x + 1) * block_width);

            unsigned long long total_r = 0, total_g = 0, total_b = 0;
            int pixel_count = 0;
            
            for (int py = start_py; py < end_py; ++py) {
                int flipped_y = (renderer->config.source_height - 1) - py;
                for (int px = start_px; px < end_px; ++px) {
                    const unsigned char* pixel = &renderer->pixel_buffer[(flipped_y * renderer->config.source_width + px) * 3];
                    total_r += pixel[0];
                    total_g += pixel[1];
                    total_b += pixel[2];
                    pixel_count++;
                }
            }

            if (pixel_count > 0) {
                unsigned char r = total_r / pixel_count;
                unsigned char g = total_g / pixel_count;
                unsigned char b = total_b / pixel_count;

                // Stateful color optimization
                uint32_t current_color = (r << 16) | (g << 8) | b;
                if (current_color != last_color) {
                    buf_ptr += sprintf(buf_ptr, "\033[38;2;%d;%d;%dm", r, g, b);
                    last_color = current_color;
                }

                // Weighted brightness for more perceptually accurate luminance
                float brightness = (0.2126f * r + 0.7152f * g + 0.0722f * b);
                int palette_index = (int)((brightness / 255.0f) * (PALETTE_COUNT - 1));
                *buf_ptr++ = ASCII_PALETTE[palette_index];

            } 
            else {
                 *buf_ptr++ = ' ';
            }
        }
        *buf_ptr++ = '\n';
    }
    // Add reset code at the end
    buf_ptr += sprintf(buf_ptr, "\033[0m");

    // Write the entire buffer in one go
    fputs(renderer->frame_buffer, stdout);
    fflush(stdout);

    // --- Update Stats ---
    double end_time = time_get_seconds();
    renderer->conversion_time_ms = (end_time - start_time) * 1000.0;
    
    if (renderer->frame_start_time > 0.0) {
        renderer->fps = 1.0 / (end_time - renderer->frame_start_time);
    }
    renderer->frame_start_time = end_time;
}

void ascii_renderer_get_stats(
    AsciiRenderer *renderer,
    double *conversion_time_ms,
    double *fps
) {
    *conversion_time_ms = renderer->conversion_time_ms;
    *fps = renderer->fps;
}