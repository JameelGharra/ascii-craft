#ifndef _queue_h_
#define _queue_h_

#include <stdbool.h>

typedef struct Queue Queue;

Queue *queue_create();
void queue_destroy(Queue *queue);

void queue_enqueue(Queue *queue, void *item);
void *queue_dequeue(Queue *queue);

bool queue_is_empty(Queue *queue);
int queue_get_size(Queue *queue);

#endif