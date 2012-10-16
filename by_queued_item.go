package main

type ByQueuedItemId []*QueuedItem

func (b ByQueuedItemId) Len() int {
	return len(b)
}

func (b ByQueuedItemId) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b ByQueuedItemId) Less(i, j int) bool {
	return b[i].id < b[j].id
}
