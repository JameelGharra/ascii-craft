#include "queue.h"
#include <stdlib.h>

typedef struct QueueNode QueueNode;

static struct QueueNode {
    void *item;
    QueueNode *next;
};

struct Queue {
    QueueNode *front;
    QueueNode *rear;
    int size;
};

Queue *queue_create() {
    Queue *queue = (Queue *)malloc(sizeof(Queue));
    if (!queue) {
        return NULL;
    }
    queue->front = NULL;
    queue->rear = NULL;
    queue->size = 0;
    return queue;
}

void queue_destroy(Queue *queue) {
    if (queue) {
        while (queue->front) {
            QueueNode *temp = queue->front;
            queue->front = queue->front->next;
            free(temp);
        }
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
    new_node->item = item;
    new_node->next = NULL;
    if (queue->rear) {
        queue->rear->next = new_node;
    } else {
        queue->front = new_node;
    }
    queue->rear = new_node;
    queue->size++;
}
void *queue_dequeue(Queue *queue) {
    if (!queue || !queue->front) {
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
    return item;
}
bool queue_is_empty(Queue *queue) {
    return !queue || queue->size == 0;
}
int queue_get_size(Queue *queue) {
    return queue ? queue->size : 0;
}