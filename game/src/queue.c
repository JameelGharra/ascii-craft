#include <stdlib.h>
#include "queue.h"
#include "tinycthread.h"

struct QueueNode {
    void *item;
    struct QueueNode *next;
};
typedef struct QueueNode QueueNode;

struct Queue {
    QueueNode *front;
    QueueNode *rear;
    int size;
    mtx_t mtx;
};

Queue *queue_create() {
    Queue *queue = (Queue *)malloc(sizeof(Queue));
    if (!queue) {
        return NULL;
    }
    queue->front = NULL;
    queue->rear = NULL;
    queue->size = 0;
    mtx_init(&queue->mtx, mtx_plain);
    return queue;
}

void queue_destroy(Queue *queue) {
    if (queue) {
        while (queue->front) {
            QueueNode *temp = queue->front;
            queue->front = queue->front->next;
            free(temp);
        }
        mtx_destroy(&queue->mtx);
        free(queue);
    }
}
void queue_enqueue(Queue *queue, void *item) {
    if (!queue) {
        return;
    }
    QueueNode *new_node = (QueueNode *)malloc(sizeof(QueueNode));
    if (!new_node) {
        return;
    }
    mtx_lock(&queue->mtx);
    new_node->item = item;
    new_node->next = NULL;
    if (queue->rear) {
        queue->rear->next = new_node;
    } else {
        queue->front = new_node;
    }
    queue->rear = new_node;
    queue->size++;
    mtx_unlock(&queue->mtx);
}
void *queue_dequeue(Queue *queue) {
    mtx_lock(&queue->mtx);
    if (!queue || !queue->front) {
        mtx_unlock(&queue->mtx);
        return NULL;
    }
    QueueNode *temp = queue->front;
    void *item = temp->item;
    queue->front = queue->front->next;
    if (!queue->front) {
        queue->rear = NULL;
    }
    free(temp);
    queue->size--;
    mtx_unlock(&queue->mtx);
    
    return item;
}
bool queue_is_empty(Queue *queue) {
    if (!queue) {
        return true;
    }
    return queue->size == 0;
}
int queue_get_size(Queue *queue) {
    if (!queue) {
        return 0;
    }
    int size = queue->size;
    return size;
}