#include "camera.h"
#include "matrix.h"

void camera_update_matrices(Camera *camera) {
    set_matrix_3d(
        camera->view_proj_matrix,
        camera->window_width, camera->window_height,
        camera->x, camera->y, camera->z,
        camera->rx, camera->ry,
        camera->fov,
        camera->ortho,
        camera->render_radius);
    frustum_planes(camera->frustum_planes, camera->render_radius, camera->view_proj_matrix);
}