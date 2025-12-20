package huffman

type PriorityQueue []*HuffmanNode

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].count < pq[j].count
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

// these to be used internally by the heap package - I keep getting confused between heap.Push and respective version of this
func (pq *PriorityQueue) Push(x any) {
	elem := x.(*HuffmanNode)
	*pq = append(*pq, elem)
}
func (pq *PriorityQueue) Pop() any {
	accessPq := *pq
	length := len(accessPq)
	elem := accessPq[length-1]
	accessPq[length-1] = nil // not necessary to be honest, but let it be
	*pq = accessPq[0 : length-1]
	return elem
}
