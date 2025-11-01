#ifndef __camera_h__
#define __camera_h__

typedef struct {
    int window_width, window_height;
    float fov;
    int ortho;
    float x, y, z;
    float rx, ry;
    int render_radius;
    float view_proj_matrix[16];
    float frustum_planes[6][4];
} Camera;

void camera_update_matrices(Camera *camera);

#endif