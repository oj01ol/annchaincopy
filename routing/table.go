package routing

import (
	//"fmt"
	"sync"
	"time"

	//"net"
	crand "crypto/rand"
	"math/rand"
	//"log"
)

const (
	alpha              = 3
	Hashlenth          = 20
	hashBits           = Hashlenth * 8
	findsize           = 16
	bucketSize         = 16
	maxFindFailures    = 5
	nBuckets           = hashBits / 15 //10
	seedCount          = 30
	seedMaxAge         = 7 * 24 * time.Hour
	maxReplacements    = 10
	bucketMinDistance  = hashBits - nBuckets //0~151,152,..,160
	refreshInterval    = 30 * time.Minute
	revalidateInterval = 30 * time.Second
)

type Table struct {
	buckets [nBuckets]*bucket
	//bucket	[]Node
	mutex sync.Mutex
	//selfID	Hash
	db       *nodeDB
	self     Node
	nursery  []Node
	net      transport
	closeReq chan struct{}
	closed   chan struct{}
	//rsp		chan Packet
}

type Hash [Hashlenth]byte

type bucket struct {
	entries      []Node
	replacements []Node
}

//node used in other modules
type Node interface {
	GetAddr() string
	GetID() Hash
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
	AddedAt() time.Times
	UpdateAddTime(time.Times) Node
	Init(id Hash, addr string) Node
}

type transport interface {
	Ping(Hash, string) error
	FindNode(addr string, target Hash) ([]Node, error)
	Close()
}

type nodesByDistance struct {
	entries []Node
	target  Hash
}

func NewTable(t transport, selfID Hash, selfAddr string, nodeDBPath string, bootnodes []Node) (*Table, error) {
	db, err := newNodeDB(nodeDBPath, selfID)
	if err != nil {
		return nil, err
	}
	var node Node
	n := node.Init(selfID, selfAddr)
	tab := &Table{
		net:      t,
		db:       db,
		self:     n,
		closeReq: make(chan struct{}),
		closed:   make(chan struct{}),
	}
	if err := tab.setFallbackNodes(bootnodes); err != nil {
		return nil, err
	}
	tab.loadSeedNodes()
	tab.db.ensureExpirer() //expire db
	go tab.loop()          //0
	return tab, nil
}

//transfer bootnodes []*Node to nursery nodes []*tNode
//remove the useless information
func (t *Table) setFallbackNodes(nodes []Node) error {
	t.nursery = make([]Node, 0, len(nodes))
	for _, n := range nodes {
		t.nursery = append(t.nursery, n)
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

func (t *Table) add(n Node) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	b := t.bucket(n.GetID())
	if !t.bumpOrAdd(b, n) {
		t.addReplacement(b, n)
	}
}

func (t *Table) bucket(id Hash) *bucket {
	d := distance(t.self.GetID(), id)
	if d <= bucketMinDistance {
		return t.buckets[0]
	}
	return t.buckets[d-bucketMinDistance-1]
}

func (t *Table) bumpOrAdd(b *bucket, n Node) bool {
	if b.bump(n) {
		return true
	}
	if len(b.entries) >= bucketSize {
		return false
	}

	node := n.UpdateAddTime(time.Now())
	b.entries = pushNode(b.entries, node, bucketSize)
	b.replacements = deleteNode(b.replacements, node)

	return true

}

func (t *Table) addReplacement(b *bucket, n Node) {
	for _, e := range b.replacements {
		if e.GetID() == n.GetID() {
			return
		}
	}
	b.replacements = pushNode(b.replacements, n, maxReplacements)
}

func (b *bucket) bump(n Node) bool {
	for i := range b.entries {
		if b.entries[i].GetID() == n.GetID() {
			copy(b.entries[1:], b.entries[:i])
			b.entries[0] = n
			return true
		}
	}
	return false
}

func pushNode(list []Node, n Node, max int) []Node {
	if len(list) < max {
		list = append(list, nil)
	}
	copy(list[1:], list)
	list[0] = n
	return list
}

func deleteNode(list []Node, n Node) []Node {
	for i := range list {
		if list[i].GetID() == n.GetID() {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (t *Table) loop() {
	var (
		revalidate     = time.NewTimer(t.nextRevalidateTime())
		refresh        = time.NewTicker(refreshInterval)
		revalidateDone = make(chan struct{})
		refreshDone    = make(chan struct{})
	)
	defer refresh.Stop()
	defer revalidate.Stop()

	go t.doRefresh(refreshDone)

loop:
	for {
		select {
		case <-refresh.C:
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				go t.doRefresh(refreshDone)
			}
		case <-refreshDone:
			refreshDone = nil
		case <-revalidate.C:
			go t.doRevalidate(revalidateDone)
		case <-revalidateDone:
			revalidate.Reset(t.nextRevalidateTime())

		case <-t.closeReq:
			break loop

		}
	}
	if t.net != nil {
		t.net.Close()
	}
	if refreshDone != nil {
		<-refreshDone
	}
	t.db.close()
	close(t.closed)

}

func (t *Table) nextRevalidateTime() time.Duration {
	return time.Duration(rand.Int63n(int64(revalidateInterval)))
}

func (t *Table) doRefresh(done chan struct{}) {
	defer close(done)
	t.GetNodeByNet(t.self.GetID())
	for i := 0; i < 3; i++ {
		var target Hash
		crand.Read(target[:])
		t.GetNodeByNet(target)
	}

}

func (t *Table) doRevalidate(done chan struct{}) {
	defer func() { done <- struct{}{} }()
	bi := rand.Intn(len(t.buckets)) //need seeds
	b := t.buckets[bi]
	if len(b.entries) == 0 {
		return
	}
	last := b.entries[len(b.entries)-1]
	if last == nil {
		return
	}
	err := t.net.ping(last.GetID(), last.GetAddr())
	t.mutex.Lock()
	defer t.mutex.Unlock()
	b = t.buckets[bi]
	if err == nil {
		b.bump(last)
		return
	}
	t.replace(b, last)

}

func (t *Table) replace(b *bucket, last Node) Node {
	if len(b.entries) == 0 || b.entries[len(b.entries)-1].GetID() != last.GetID() {
		return nil
	}
	if len(b.replacements) == 0 {
		b.entries = deleteNode(b.entries, last)
		return nil
	}
	r := b.replacements[rand.Intn(len(b.replacements))]
	b.entries[len(b.entries)-1] = r
	return r
}

//get node address by Nodeid
//1.find in the bucket
//2.findnode in the network

func (t *Table) GetNodeLocally(targetID Hash) []Node {
	t.mutex.Lock()
	result := t.closest(targetID, findsize)
	t.mutex.Unlock()

	return result.entries
}

func (t *Table) GetNodeByNet(targetID Hash) []Node {
	var (
		asked          = make(map[Hash]bool)
		result         *nodesByDistance
		seen           = make(map[Hash]bool)
		reply          = make(chan []Node, alpha)
		pendingQueries = 0
	)

	asked[t.self.GetID()] = true

	t.mutex.Lock()
	result = t.closest(targetID, findsize)
	t.mutex.Unlock()

	for {
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.GetID()] {
				asked[n.GetID()] = true
				seen[n.GetID()] = true
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
			if n != nil && !seen[n.GetID()] {
				seen[n.GetID()] = true
				result.push(n, findsize)
			}
		}
		pendingQueries--
	}

	return result.entries

}

//ask n for the Node info
func (t *Table) findNode(n Node, targetID Hash, reply chan<- []Node) {

	//send and receive simulation
	fails := t.db.findFails(n.GetID())
	var findrsp findResponse

	r, err := t.net.FindNode(n.GetAddr(), targetID)
	//handle data
	//errjson := json.Unmarshal(r, &findrsp)
	if err != nil || len(r) == 0 {
		fails++
		t.db.updateFindFails(n.GetID(), fails)
		if fails >= maxFindFailures {
			t.delete(n)
		}
	} else if fails > 0 {
		t.db.updateFindFails(n.GetID(), fails-1)
	}

	for _, n := range r {
		t.add(n)
	}
	reply <- r
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

func (h *nodesByDistance) push(n Node, maxElems int) {
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
