package routing

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	crand "crypto/rand"
	"encoding/binary"
	"math/rand"

	"github.com/pkg/errors"
)

var (
	HASH_FORMAT_ERR = errors.New("hash format err")
)

type Hash []byte
type HashKey = string

func ToHash(ori []byte) Hash {
	var h Hash
	h.Copy(ori)
	return h
}

func NewHash() Hash {
	h := make([]byte, c.HashLength)
	for i := range h {
		h[i] = 0xff
	}
	return h
}

func (h Hash) Check() error {
	if len(h) != c.HashLength {
		return HASH_FORMAT_ERR
	}
	hex := hex.EncodeToString(h)
	if len(hex) != c.HashLength*2 {
		return HASH_FORMAT_ERR
	}
	return nil
}

func (h *Hash) Copy(ch []byte) {
	(*h) = make([]byte, c.HashLength)
	copy(*h, ch)
	i := len(ch)
	for ; i < c.HashLength; i++ {
		(*h)[i] = 0xff
	}
}

func (h Hash) Equal(hh []byte) bool {
	return bytes.Equal(h, hh)
}

func (h Hash) AsKey() HashKey {
	return HashKey(h)
}

type bucket struct {
	entries      []*Node
	replacements []*Node
}

// TODO use buffer
func (b *bucket) String() string {
	str := "entries"
	for i := range b.entries {
		if i == 0 {
			str = fmt.Sprintf("%v\n:node[%v]:%v", str, i, b.entries[i].String())
		} else {
			str = fmt.Sprintf("%v,node[%v]:%v", str, i, b.entries[i].String())
		}
	}
	str = fmt.Sprintf("%v\nreplacements:\n", str)
	for i := range b.replacements {
		str = fmt.Sprintf("%v,node[%v]:%v", str, i, b.entries[i].String())
	}
	return str
}

//node used in other modules
type Node struct {
	Time int64
	ID   Hash
	Addr string
}

func NewNode(id Hash, addr string) *Node {
	n := &Node{
		Addr: addr,
	}
	n.ID.Copy(id)
	return n
}

func (n *Node) Equal(nn *Node) bool {
	return n.Time == nn.Time && n.ID.Equal(nn.ID) && n.Addr == nn.Addr
}

func (n *Node) Marshal() ([]byte, error) {
	return json.Marshal(n)
}

func (n *Node) Unmarshal(bys []byte) error {
	if n == nil {
		n = &Node{}
	}
	return json.Unmarshal(bys, n)
}

func (n *Node) AddedAt() time.Time {
	return time.Unix(n.Time, 0)
}

func (n *Node) UpdateAddTime(time time.Time) {
	n.Time = time.Unix()
}

func (n *Node) GetAddr() string {
	return n.Addr
}

func (n *Node) GetID() Hash {
	return n.ID
}

func (n *Node) String() string {
	return fmt.Sprintf("ID:%x,Addr:%v,Time:%v", n.ID, n.Addr, n.Time)
}

func (n *Node) MarshalJSON() ([]byte, error) {
	st := struct {
		Time string
		ID   string
		Addr string
	}{
		Time: time.Unix(n.Time, 0).Format(time.RFC3339),
		ID:   fmt.Sprintf("%x", n.ID),
		Addr: n.Addr,
	}
	return json.Marshal(&st)
}

func (n *Node) UnmarshalJSON(bys []byte) error {
	st := struct {
		Time string
		ID   string
		Addr string
	}{}
	if err := json.Unmarshal(bys, &st); err != nil {
		return err
	}
	idbys, err := hex.DecodeString(st.ID)
	if err != nil {
		return err
	}
	n.ID.Copy(idbys)
	ttm, err := time.Parse(time.RFC3339, st.Time)
	if err != nil {
		return err
	}
	n.Time = ttm.Unix()
	n.Addr = st.Addr
	return nil
}

type INode interface {
	GetAddr() string
	GetID() Hash
}

type transport interface {
	Ping(addr string) error
	FindNode(addr string, target Hash) ([]INode, error)
}

type nodesByDistance struct {
	entries []*Node
	target  Hash
}

type Table struct {
	buckets []*bucket
	//bucket	[]Node
	mutex sync.Mutex
	//selfID	Hash
	db       *nodeDB
	self     *Node
	nursery  []*Node
	net      transport
	rand     *rand.Rand
	closeReq chan struct{}
	closed   chan struct{}

	//rsp		chan Packet
}

func NewTable(t transport, selfID Hash, selfAddr string, nodeDBPath string, bootnodes []INode) (*Table, error) {
	if err := selfID.Check(); err != nil {
		return nil, err
	}

	db, err := newNodeDB(nodeDBPath, selfID)
	if err != nil {
		return nil, err
	}
	n := NewNode(selfID, selfAddr)
	tab := &Table{
		buckets:  make([]*bucket, c.nBuckets),
		net:      t,
		db:       db,
		self:     n,
		rand:     rand.New(rand.NewSource(0)),
		closeReq: make(chan struct{}),
		closed:   make(chan struct{}),
	}
	if err := tab.setFallbackNodes(_inodesToNodes(bootnodes)); err != nil {
		return nil, err
	}
	for i := range tab.buckets {
		tab.buckets[i] = &bucket{}
	}
	tab.seedRand()
	tab.loadSeedNodes()
	tab.db.ensureExpirer() //expire db
	return tab, nil
}

func (t *Table) Start() {
	go t.loop()
}

func (t *Table) Stop() {
	t.closeReq <- struct{}{}
}

func (t *Table) String() string {
	//str := "buckets"
	//for i := range t.buckets {
	//	if i == 0 {
	//		str = fmt.Sprintf("%v:\nbuckets[%v]:%v\n", str, i, t.buckets[i].String())
	//	} else {
	//		str = fmt.Sprintf("%v,buckets[%v]:%v\n", str, i, t.buckets[i].String())
	//	}
	//}
	//return str

	type stBucks struct {
		Entries      []*Node
		Replacements []*Node
	}
	itab := struct {
		Buckets []stBucks
	}{}
	itab.Buckets = make([]stBucks, len(t.buckets))
	for i := range itab.Buckets {
		itab.Buckets[i].Entries = t.buckets[i].entries
		itab.Buckets[i].Replacements = t.buckets[i].replacements
	}
	jsonBytes, _ := json.Marshal(&itab)
	return string(jsonBytes)
}

func (t *Table) buketsCount() []int {
	slc := make([]int, len(t.buckets))
	for i := range t.buckets {
		slc[i] = len(t.buckets[i].entries)
	}
	return slc
}

//transfer bootnodes []*Node to nursery nodes []*tNode
//remove the useless information
func (t *Table) setFallbackNodes(nodes []*Node) error {
	t.nursery = make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		t.nursery = append(t.nursery, n)
	}
	return nil
}

func (t *Table) loadSeedNodes() {
	seeds := t.db.querySeeds(c.seedCount, c.seedMaxAge) //if reboot after one week, will get nothing
	seeds = append(seeds, t.nursery...)
	for i := range seeds {
		seed := seeds[i]
		t.add(seed)
	}
}

func (t *Table) add(n *Node) error {
	if t.self.GetID().Equal(n.GetID()) {
		return nil
	}
	if n.InComplete() {
		return errors.New("add node incomplete")
	}
	result := t.closest(n.GetID(), 1)
	if len(result.entries) != 0 {
		dutyn := result.entries[0]
		if distance(t.self.GetID(), n.GetID()) <= distance(dutyn.GetID(), n.GetID()) {
			t.mutex.Lock()
			nb := t.bucket(n.GetID())
			if len(nb.entries) >= 2*c.bucketSize {
				t.addReplacement(nb, n)
			} else {
				n.UpdateAddTime(time.Now())
				nb.entries = pushNode(nb.entries, n, 2*c.bucketSize)
				nb.replacements = deleteNode(nb.replacements, n)
			}
			t.mutex.Unlock()
			return nil
		}
	}

	t.mutex.Lock()
	b := t.bucket(n.GetID())
	if !t.bumpOrAdd(b, n) {
		t.addReplacement(b, n)
	}
	t.mutex.Unlock()
	return nil
}

func (n *Node) InComplete() bool {
	return n.GetAddr() == "" || n.GetID().Equal(Hash{})
}

func (t *Table) bucket(id Hash) *bucket {
	d := distance(t.self.GetID(), id)
	if d <= c.bucketMinDistance {
		return t.buckets[0]
	}
	return t.buckets[d-c.bucketMinDistance-1]
}

func (t *Table) bumpOrAdd(b *bucket, n *Node) bool {
	if b.bump(n) {
		return true
	}
	if len(b.entries) >= c.bucketSize {
		return false
	}

	n.UpdateAddTime(time.Now())
	b.entries = pushNode(b.entries, n, c.bucketSize)
	b.replacements = deleteNode(b.replacements, n)

	return true

}

func (t *Table) addReplacement(b *bucket, n *Node) {
	for _, e := range b.replacements {
		if e.GetID().Equal(n.GetID()) {
			return
		}
	}
	b.replacements = pushNode(b.replacements, n, c.maxReplacements)
}

func (b *bucket) bump(n *Node) bool {
	for i := range b.entries {
		if b.entries[i].GetID().Equal(n.GetID()) {
			copy(b.entries[1:], b.entries[:i])
			b.entries[0] = n
			return true
		}
	}
	return false
}

func pushNode(list []*Node, n *Node, max int) []*Node {
	if len(list) < max {
		list = append(list, nil)
	}
	copy(list[1:], list)
	list[0] = n
	return list
}

func deleteNode(list []*Node, n *Node) []*Node {
	for i := range list {
		if list[i].GetID().Equal(n.GetID()) {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (t *Table) seedRand() {
	var b [8]byte
	crand.Read(b[:])

	t.mutex.Lock()
	t.rand.Seed(int64(binary.BigEndian.Uint64(b[:])))
	t.mutex.Unlock()
}

func (t *Table) loop() {
	var (
		revalidate     = time.NewTimer(t.nextRevalidateTime())
		refresh        = time.NewTicker(c.refreshInterval)
		revalidateDone = make(chan struct{})
		refreshDone    = make(chan struct{})
	)

	go t.doRefresh(refreshDone)

loop:
	for {
		select {
		case <-refresh.C:
			t.seedRand()
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
	if refreshDone != nil {
		<-refreshDone
	}
	refresh.Stop()
	revalidate.Stop()
	t.db.close()
	close(t.closed)

}

func (t *Table) nextRevalidateTime() time.Duration {
	return time.Duration(t.rand.Int63n(int64(c.revalidateInterval)))
}

func (t *Table) doRefresh(done chan struct{}) {
	t.doRefreshCallback(done, nil)
}

func (t *Table) doRefreshCallback(done chan struct{}, deal func(addr string)) {
	t.GetNodeByNetCallback(t.self.GetID(), deal)
	for i := 0; i < 3; i++ {
		target := NewHash()
		crand.Read(target[:])
		t.GetNodeByNetCallback(target, deal)
	}
	close(done)
}

func (t *Table) doRevalidate(done chan struct{}) {
	defer func() { done <- struct{}{} }()
	bi := t.rand.Intn(len(t.buckets)) //need seeds
	b := t.buckets[bi]
	if len(b.entries) == 0 {
		return
	}
	last := b.entries[len(b.entries)-1]
	if last == nil {
		return
	}
	err := t.net.Ping(last.GetAddr())
	t.mutex.Lock()
	defer t.mutex.Unlock()
	b = t.buckets[bi]
	if err == nil {
		t.db.updateLastPongReceived(last.GetID(), time.Now())
		b.bump(last)
		return
	}
	t.replace(b, last)

}

func (t *Table) replace(b *bucket, last *Node) *Node {
	if len(b.entries) == 0 || !b.entries[len(b.entries)-1].GetID().Equal(last.GetID()) {
		return nil
	}
	if len(b.replacements) == 0 {
		b.entries = deleteNode(b.entries, last)
		return nil
	}
	r := b.replacements[t.rand.Intn(len(b.replacements))]
	b.replacements = deleteNode(b.replacements, r)
	b.entries[len(b.entries)-1] = r
	return r
}

//get node address by Nodeid
//1.find in the bucket
//2.findnode in the network

func (t *Table) GetNodeLocally(targetID Hash) []INode {
	t.mutex.Lock()
	result := t.closest(targetID, c.findsize)
	t.mutex.Unlock()

	return _nodesToINodes(result.entries)
}

func _nodesToINodes(nodes []*Node) []INode {
	inodes := make([]INode, len(nodes))
	for i := range inodes {
		inodes[i] = nodes[i]
	}
	return inodes
}

func _inodesToNodes(inodes []INode) []*Node {
	nodes := make([]*Node, len(inodes))
	for i := range inodes {
		nodes[i] = NewNode(inodes[i].GetID(), inodes[i].GetAddr())
	}
	return nodes
}

func (t *Table) GetNodeAddr(targetID Hash) string {
	nodes := t.GetNodeByNet(targetID)
	for _, node := range nodes {
		if targetID.Equal(node.GetID()) {
			return node.GetAddr()
		}
	}
	return ""
}

func (t *Table) OnReceiveReq(node INode) error {
	n := NewNode(node.GetID(), node.GetAddr())
	return t.add(n)
}

func (t *Table) GetNodeByNet(targetID Hash) []*Node {
	return t.GetNodeByNetCallback(targetID, nil)
}

func (t *Table) GetNodeByNetCallback(targetID Hash, deal func(addr string)) []*Node {
	var (
		asked          = make(map[HashKey]bool)
		result         *nodesByDistance
		seen           = make(map[HashKey]bool)
		reply          = make(chan []*Node, c.alpha)
		pendingQueries = 0
	)

	asked[t.self.GetID().AsKey()] = true

	t.mutex.Lock()
	result = t.closest(targetID, c.findsize)
	t.mutex.Unlock()

	for {
		for i := 0; i < len(result.entries) && pendingQueries < c.alpha; i++ {
			n := result.entries[i]
			nodeKey := n.GetID().AsKey()
			if !asked[nodeKey] {
				asked[nodeKey] = true
				seen[nodeKey] = true
				pendingQueries++

				go t.findNodeCallback(n, targetID, reply, deal)
			}
		}
		if pendingQueries == 0 {
			// we have asked all closest nodes, stop the search
			break
		}
		// wait for the next reply
		for _, n := range <-reply {
			if n != nil {
				nodeKey := n.GetID().AsKey()
				if !seen[nodeKey] {
					seen[nodeKey] = true
					result.push(n, c.findsize)
				}
			}
		}
		pendingQueries--
	}

	return result.entries

}

func (t *Table) findNode(n *Node, targetID Hash, reply chan<- []*Node) {
	t.findNodeCallback(n, targetID, reply, nil)
}

//ask n for the *Node info
func (t *Table) findNodeCallback(n *Node, targetID Hash, reply chan<- []*Node, deal func(addr string)) {
	r, err := t.net.FindNode(n.GetAddr(), targetID)
	fails := t.db.findFails(n.GetID())

	if err != nil || len(r) == 0 {
		fails++
		t.db.updateFindFails(n.GetID(), fails)
		if fails >= c.maxFindFailures {
			t.delete(n)
		}
	} else if fails > 0 {
		t.db.updateLastPongReceived(n.GetID(), time.Now())
		t.db.updateFindFails(n.GetID(), fails-1)
	}

	nodes := make([]*Node, len(r))
	for i := range r {
		nodes[i] = NewNode(r[i].GetID(), r[i].GetAddr())
		t.add(nodes[i])
	}
	if deal != nil {
		deal(n.GetAddr())
	}
	reply <- nodes
}

func (t *Table) delete(n *Node) {
	t.mutex.Lock()
	b := t.bucket(n.GetID())
	b.entries = deleteNode(b.entries, n)
	t.mutex.Unlock()
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
