package routing

import (
	crand "crypto/rand"
	"math/rand"
	"sort"
	"testing"
	"time"
)

func genSortNodeForTest(dis int) SortNode {
	return SortNode{
		dis:  dis,
		Node: &Node{Time: int64(dis)},
	}
}

func genNodeForTest() *Node {
	node := &Node{}
	node.ID = NewHash()
	crand.Read(node.ID)
	return node
}

func popAndCheckRes(t *testing.T, seq int, sh *SortNodeHeap, size int, sortSlc sort.IntSlice) {
	if sh.Len() != size {
		t.Error("unexpected size", seq)
		return
	}
	sort.Sort(sortSlc)
	sortSlc = sortSlc[:size]
	var i int
	for sh.Len() != 0 {
		if i >= len(sortSlc) {
			t.Error("out of length of sortslc", seq)
			return
		}
		popn, ok := sh.PopNode()
		if !ok {
			t.Error("len() wrong, pop no node", seq)
			return
		}
		curDis := popn.dis
		if sortSlc[len(sortSlc)-i-1] != curDis {
			t.Error("seq:", seq, ",unexpected result:", curDis, ",should:", sortSlc[i])
			return
		}
		i++
	}
}

func TestSortNodeHeap(t *testing.T) {
	const (
		HEAP_SIZE = 10
	)

	rd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// num of nodes is equal to heap size
	sh := &SortNodeHeap{}
	sh.Init(HEAP_SIZE, nil)
	sortSlc := make(sort.IntSlice, HEAP_SIZE)
	for i := range sortSlc {
		sortSlc[i] = rd.Intn(666)
		sh.PushSNode(genSortNodeForTest(sortSlc[i]))
	}
	popAndCheckRes(t, 1, sh, HEAP_SIZE, sortSlc)

	// num of nodes is smaller than heap size
	sh.Init(HEAP_SIZE-5, nil)
	sortSlc = make(sort.IntSlice, HEAP_SIZE-5)
	for i := range sortSlc {
		sortSlc[i] = rd.Intn(666)
		sh.PushSNode(genSortNodeForTest(sortSlc[i]))
	}
	popAndCheckRes(t, 2, sh, HEAP_SIZE-5, sortSlc)

	// num of nodes is bigger than heap size
	sh.Init(HEAP_SIZE, nil)
	sortSlc = make(sort.IntSlice, HEAP_SIZE+100)
	for i := range sortSlc {
		sortSlc[i] = rd.Intn(666)
		sh.PushSNode(genSortNodeForTest(sortSlc[i]))
	}
	popAndCheckRes(t, 3, sh, HEAP_SIZE, sortSlc)

	// has same distance
	sh.Init(HEAP_SIZE, nil)
	sortSlc = make(sort.IntSlice, HEAP_SIZE)
	for i := range sortSlc {
		sortSlc[i] = rd.Intn(HEAP_SIZE - 5)
		sh.PushSNode(genSortNodeForTest(sortSlc[i]))
	}
	popAndCheckRes(t, 4, sh, HEAP_SIZE, sortSlc)

	// getMin method
	sh.Init(HEAP_SIZE, nil)
	sortSlc = make(sort.IntSlice, HEAP_SIZE)
	for i := range sortSlc {
		sortSlc[i] = rd.Intn(666)
		sh.PushSNode(genSortNodeForTest(sortSlc[i]))
	}
	sort.Sort(sortSlc)
	for i := 0; i < 3; i++ {
		res, resi := sh.GetMin()
		shouldi := i
		if sortSlc[shouldi] != res.dis {
			t.Error("not minimum node", res.dis, ",should:", sortSlc[shouldi])
			sh.Print()
			return
		}
		rmv, ok := sh.Remove(resi)
		if !ok || rmv.dis != res.dis {
			t.Error("remove wrong", ok, rmv.dis, ",should:", res.dis)
			return
		}
	}
	popAndCheckRes(t, 5, sh, HEAP_SIZE-3, sortSlc[3:])
}

func TestNodsByDisAndSortHeap(t *testing.T) {
	const (
		HEAP_SIZE = 10
	)
	targetID := NewHash()

	nodeByDis := &nodesByDistance{target: targetID}

	sh := &SortNodeHeap{}
	sh.Init(HEAP_SIZE, targetID)

	for i := 0; i < HEAP_SIZE*10; i++ {
		node := genNodeForTest()
		sh.PushNode(node)
		nodeByDis.push(node, HEAP_SIZE)
	}

	if sh.size != HEAP_SIZE {
		t.Error("wrong heap size,want:", HEAP_SIZE, ",in real:", sh.size)
		return
	}
	if len(nodeByDis.entries) != HEAP_SIZE {
		t.Error("wrong nodebydis size,want:", HEAP_SIZE, ",in real:", len(nodeByDis.entries))
		return
	}

	disCountsH := make(map[int]int) // map[distance]count
	for _, nodes := range sh.ToNodeSlc() {
		disCountsH[distance(nodes.ID, sh.targetID)] += 1
	}
	disCountsD := make(map[int]int) // map[distance]count
	for i := range nodeByDis.entries {
		nd := nodeByDis.entries[i]
		disCountsD[distance(nd.ID, nodeByDis.target)] += 1
	}
	// different sort methods may result to different id set,
	// but the counts of different distances are the same
	for k, v := range disCountsD {
		if disCountsH[k] != v {
			t.Error("different distance count between nodeByDis and nodeHeap")
			return
		}
	}
}

var sortBenchSlc []*Node
var sortTarget Hash

const (
	SORT_TABLE_SIZE = 16
	BENCH_DATA_SIZE = SORT_TABLE_SIZE * 3
)

func init() {
	Init(NewConfigurable())
	sortBenchSlc = make([]*Node, BENCH_DATA_SIZE)
	for i := range sortBenchSlc {
		sortBenchSlc[i] = genNodeForTest()
	}
	sortTarget = NewHash()
	crand.Read(sortTarget)
}

func BenchmarkPushNodeHeap(b *testing.B) {
	sh := &SortNodeHeap{}
	sh.Init(SORT_TABLE_SIZE, sortTarget)
	for i := 0; i < b.N; i++ {
		sh.PushNode(sortBenchSlc[i%len(sortBenchSlc)])
	}
}

func BenchmarkPushNodeOrig(b *testing.B) {
	h := &nodesByDistance{target: sortTarget}
	for i := 0; i < b.N; i++ {
		h.push(sortBenchSlc[i%len(sortBenchSlc)], SORT_TABLE_SIZE)
	}
}
