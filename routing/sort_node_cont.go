package routing

import (
	"container/heap"
	"fmt"
)

type SortNode struct {
	dis int
	*Node
}

func (sn *SortNode) String() string {
	return fmt.Sprintf("dis:%v", sn.dis)
}

// keep size entries that have minimum distance
type SortNodeHeap struct {
	entries  []SortNode
	size     int
	targetID Hash
}

func (snh *SortNodeHeap) Init(size int, targetID Hash) {
	snh.entries = make([]SortNode, size)
	snh.targetID = targetID
}

func (snh SortNodeHeap) Len() int {
	return snh.size
}

func (snh SortNodeHeap) Less(i, j int) bool {
	return snh.entries[i].dis >= snh.entries[j].dis
}

func (snh SortNodeHeap) Swap(i, j int) {
	snh.entries[i], snh.entries[j] = snh.entries[j], snh.entries[i]
}

func (snh *SortNodeHeap) Push(x interface{}) {
	if snh.size == len(snh.entries) {
		return
	}
	snh.entries[snh.size] = x.(SortNode)
	snh.size++
}

func (snh *SortNodeHeap) PushNode(n *Node) {
	sn := SortNode{
		dis:  distance(n.GetID(), snh.targetID),
		Node: n,
	}
	snh.PushSNode(sn)
}

func (snh *SortNodeHeap) PushSNode(sn SortNode) {
	if snh.size == len(snh.entries) {
		if snh.entries[0].dis < sn.dis {
			return
		}
		snh.PopNode()
	}
	heap.Push(snh, sn)
}

func (snh *SortNodeHeap) PopNode() (sn SortNode, ok bool) {
	r := heap.Pop(snh)
	if r != nil {
		sn = r.(SortNode)
		ok = true
	}
	return
}

func (snh *SortNodeHeap) ToINodeSlc() []INode {
	inodes := make([]INode, snh.size)
	for i := 0; i < snh.size; i++ {
		inodes[i] = snh.entries[i].Node
	}
	return inodes
}

func (snh *SortNodeHeap) ToNodeSlc() []*Node {
	inodes := make([]*Node, snh.size)
	for i := 0; i < snh.size; i++ {
		inodes[i] = snh.entries[i].Node
	}
	return inodes
}

func (snh *SortNodeHeap) Exec(exec func(n *Node) bool) bool {
	for i := 0; i < snh.size; i++ {
		if !exec(snh.entries[i].Node) {
			return false
		}
	}
	return true
}

func (snh *SortNodeHeap) Full() bool {
	return snh.size == len(snh.entries)
}

// Pop() pops the node with minimum distance
func (snh *SortNodeHeap) Pop() interface{} {
	if snh.size == 0 {
		return nil
	}
	ret := snh.entries[snh.size-1]
	snh.entries[snh.size-1] = SortNode{}
	snh.size--
	return ret

}

func (snh *SortNodeHeap) GetMin() (sn SortNode, minIndex int) {
	if snh.size == 0 {
		return sn, -1
	}
	if snh.size == 1 {
		return snh.entries[0], 0
	}
	lastP := (snh.size)>>2 - 1
	minIndex = snh.size - 1
	min := snh.entries[minIndex].dis
	for i := minIndex; i > lastP; i-- {
		if snh.entries[i].dis < min {
			minIndex = i
			min = snh.entries[minIndex].dis
		}
	}
	sn = snh.entries[minIndex]
	return
}

func (snh *SortNodeHeap) Remove(i int) (sn SortNode, ok bool) {
	r := heap.Remove(snh, i)
	if r != nil {
		sn = r.(SortNode)
		ok = true
	}
	return
}

func (snh *SortNodeHeap) Print() {
	fmt.Println("heap:", snh.entries, ",size:", snh.size)
}

////////////////////////////////////////////////////////////////

type NodeRoundQueue struct {
	nodes       []*Node
	front, tail int
}

func NewNodeRoundQueue(size int) *NodeRoundQueue {
	return &NodeRoundQueue{
		nodes: make([]*Node, size+1),
	}
}

func (l *NodeRoundQueue) Push(n *Node) {
	if l.Full() {
		l.Out()
	}
	l.nodes[l.tail] = n
	l.tail = (l.tail + 1) % len(l.nodes)
}

func (l *NodeRoundQueue) Len() int    { return (l.tail + len(l.nodes) - l.front) % len(l.nodes) }
func (l *NodeRoundQueue) Full() bool  { return (l.tail+1)%len(l.nodes) == l.front }
func (l *NodeRoundQueue) Empty() bool { return l.front == l.tail }
func (l *NodeRoundQueue) Reset()      { l.tail = 0; l.front = 0 }
func (l *NodeRoundQueue) Out() (*Node, bool) {
	if l.Empty() {
		return nil, false
	}
	v := l.nodes[l.front]
	l.front = (l.front + 1) % len(l.nodes)
	return v, true
}

func (l *NodeRoundQueue) DataSlc() []*Node {
	if l.front <= l.tail {
		return l.nodes[l.front:l.tail]
	}
	return append(l.nodes[:l.tail], l.nodes[l.front:]...)
}

////////////////////////////////////////////////////////////////

type nodesByDistance struct {
	entries []*Node
	target  Hash
}

func (h *nodesByDistance) push(n *Node, maxElems int) {
	h.entries = append(h.entries, n)
	for i, node := range h.entries {
		if distance(node.GetID(), h.target) > distance(n.GetID(), h.target) {
			copy(h.entries[i+1:], h.entries[i:])
			h.entries[i] = n
			break
		}
	}
	if len(h.entries) > maxElems {
		h.entries = h.entries[:maxElems]
	}
}
