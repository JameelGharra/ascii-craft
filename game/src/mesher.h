#ifndef _mesher_h_
#define _mesher_h_

#include "map.h"
#include "sign.h"
#include <GL/glew.h>

typedef struct {
    Map *block[3][3];
    Map *light[3][3];
    int p, q;
} MesherInput;

typedef struct {
    GLfloat *data;
    int faces;
    int miny;
    int maxy;
    //signs
    GLfloat *sign_data;
    int sign_faces;
} MesherOutput;

MesherOutput *mesher_compute_chunk(MesherInput *input);
void mesher_generate_sign_mesh(MesherOutput *mesher_output, SignList *signs);
void mesher_free_output(MesherOutput **output);
void mesher_free_output_data(MesherOutput *output);

#endif