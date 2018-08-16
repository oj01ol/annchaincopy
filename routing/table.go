package routing

import (
	//"fmt"
	"sync"
	"time"
	//"net"
	//"math/rand"
	//"log"
)

const (
	alpha = 3
	Hashlenth = 20
	hashBits = Hashlenth * 8
	findsize = 16
	bucketSize = 16
	nBuckets = hashBits / 15  //10
	seedCount = 30
	seedMaxAge = 7 * 24 * time.Hour
	maxReplacements = 10
	bucketMinDistance = hashBits - nBuckets //0~151,152,..,160
)

type Table struct {
	buckets	[nBuckets]*bucket
	//bucket	[]Node
	mutex	sync.Mutex
	//selfID	Hash
	db		*nodeDB
	self	*tNode
	nursery	[]*tNode
}

type Hash [Hashlenth]byte

type bucket struct {
	entries	[]*tNode
	replacements	[]*tNode
}

//node used in other modules
type Node interface {
	GetAddr()	string
	GetID()	Hash
}

type nodesByDistance struct {
	entries []*tNode
	target	Hash
}


func NewTable(selfID Hash, selfAddr string, nodeDBPath string, bootnodes []*Node) (*Table , error){
	db, err := newNodeDB(nodeDBPath, selfID)
	if err != nil {
		return nil, err
	}
	tab := &Table{
		db:		db,
		self:	NewNode(selfID, selfAddr),
		
	}
	if err := tab.setFallbackNodes(bootnodes); err != nil {
		return nil, err
	}
	tab.loadSeedNodes()
	tab.db.ensureExpirer()
	go tab.loop()	//0
	return tab,nil
}

//transfer bootnodes []*Node to nursery nodes []*tNode
//remove the useless information
func (t *Table) setFallbackNodes(nodes []*Node) error {
	t.nursery = make([]*tNode, 0, len(nodes))
	for _, n := range nodes {
		tnode := NewNode(n.GetID(), n.GetAddr())
		t.nursery = append(t.nursery, tnode)
	}
	return nil
}


func (t *Table) loadSeedNodes() {
	seeds := t.db.querySeeds(seedCount, seedMaxAge)
	seeds = append(seeds, t.nursery...)
	for i := range seeds {
		seed := seeds[i]
		t.add(seed)
	}
}

func (t *Table) add(n *tNode) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	b := t.bucket(n.ID)
	if !t.bumpOrAdd(b, n) {
		t.addReplacement(b, n)
	}
}

func (t *Table) bucket(id Hash) *bucket {
	d := distance(t.self.ID, id)
	if d <= bucketMinDistance {
		return t.buckets[0]
	}
	return t.buckets[d-bucketMinDistance-1]
}

func (t *Table) bumpOrAdd(b *bucket, n *tNode) bool {
	if b.bump(n) {	
		return true
	}
	if len(b.entries) >= bucketSize {
		return false
	}
	n.AddTime = time.Now().Unix()
	b.entries = pushNode(b.entries, n, bucketSize)	
	b.replacements = deleteNode(b.replacements, n)	

	return true
	
}

func (t *Table) addReplacement(b *bucket, n *tNode) {
	for _, e := range b.replacements {
		if e.ID == n.ID {
			return 
		}
	}
	b.replacements = pushNode(b.replacements, n, maxReplacements)
}

func (b *bucket) bump(n *tNode) bool {
	for i:= range b.entries {
		if b.entries[i].ID == n.ID {
			copy(b.entries[1:],b.entries[:i])
			b.entries[0] = n
			return true
		}
	}
	return false
}

func pushNode(list []*tNode, n *tNode, max int) []*tNode {
	if len(list) < max {
		list = append(list, nil)
	}
	copy(list[1:],list)
	list[0] = n
	return list
}

func deleteNode(list []*tNode, n *tNode) []*tNode{
	for i := range list {
		if list[i].ID == n.ID {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (t *Table) loop() {
	
	//do refresh
}



//get node address by Nodeid
//1.find in the bucket
//2.findnode in the network

func (t *Table) GetNodeLocal(targetID Hash) string {
	t.mutex.Lock()
	result := t.closest(targetID, findsize)
	t.mutex.Unlock()
	for _,node := range result.entries {
		if node.ID == targetID{
			return node.Addr
		}
	}
	return nil
}


func (t *Table)	GetNodeNet(targetID Hash) string {
	var (
		asked	= make(map[Hash]bool)
		result	*nodesByDistance
		seen	= make(map[Hash]bool)
		reply	= make(chan []*tNode, alpha)
		pendingQueries = 0
	)
	
	asked[t.self.ID] = true
	
	t.mutex.Lock()
	result = t.closest(targetID, findsize)
	t.mutex.Unlock()
	
	for {
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.ID] {
				asked[n.ID] = true
				seen[n.ID] = true
				pendingQueries++
				
				go t.findNode(n, targetID, reply)
			}
		}
		if pendingQueries == 0 {
			// we have asked all closest nodes, stop the search
			break
		}
		// wait for the next reply
		for _, n := range <-reply {
			if n != nil && !seen[n.ID] {
				seen[n.ID] = true
				result.push(n, findsize)
			}
		}
		pendingQueries--
	}
	
	for _,node := range result.entries {
		if node.ID == targetID{
			return node.Addr
		}
	}
	return nil
	
}

//ask n for the Node info
func (t *Table) findNode(n *tNode, targetID Hash, reply chan<- []*tNode) {
	
	//send and receive simulation
	nodes := make([]Node, 0, findsize)
	/*
	for i := 0; i < findsize; i++ {
		x:= rand.Intn(100)
		node := &Node{address:"ha",ID:Hash(x)}
		nodes = append(nodes , node)
	}
	*/
	reply <- nodes
}

func (t *Table) closest(target Hash, nresults int) *nodesByDistance {
	closeSet := &nodesByDistance{target: target}
	//search all buckets
	for _, b := range t.buckets {
		for _, n := range b.entries {
			if n != nil {
				closeSet.push(n, nresults)
			}	
		}
	}

	return closeSet
}

func (h *nodesByDistance) push(n *tNode, maxElems int) {
	h.entries = append(h.entries,n)
	for i , node := range h.entries{
		if distance(node.ID , h.target) > distance(n.ID , h.target) {
			copy(h.entries[i+1:],h.entries[i:])
			h.entries[i] = n
			break
		}
	}
	if len(h.entries) > maxElems {
		h.entries = h.entries[:maxElems]
	}
}

func distance(a Hash, b Hash) int {
	lz := 0
	for i := range a {
		x := a[i] ^ b[i]
		if x == 0 {
			lz += 8
		} else {
			lz += lzcount[x]
			break
		}
	}
	return len(a)*8 - lz
	
}

var lzcount = [256]int{
	8, 7, 6, 6, 5, 5, 5, 5,
	4, 4, 4, 4, 4, 4, 4, 4,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
}

