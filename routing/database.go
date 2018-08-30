package routing

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"

	//"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	nodeDBNilHash        = Hash{}         // Special node ID to use as a nil element.
	nodeDBNodeExpiration = 24 * time.Hour // Time after which an unseen node should be dropped.
	nodeDBCleanupCycle   = time.Hour      // Time period for running the expiration task.

	nodeDBItemPrefix = []byte("n:") // Identifier to prefix node entries

	nodeDBDiscoverRoot      = ":discover"
	nodeDBDiscoverPing      = nodeDBDiscoverRoot + ":lastping"
	nodeDBDiscoverPong      = nodeDBDiscoverRoot + ":lastpong"
	nodeDBDiscoverFindFails = nodeDBDiscoverRoot + ":findfail"
)

type nodeDB struct {
	lvl    *leveldb.DB
	self   Hash
	runner sync.Once // Ensures we can start at most one expirer
	quit   chan struct{}
}

func newNodeDB(path string, self Hash) (*nodeDB, error) {
	var db *leveldb.DB
	var err error
	if path == "" {
		db, err = leveldb.Open(storage.NewMemStorage(), nil)
	} else {
		db, err = leveldb.OpenFile(path, nil)
	}

	if err != nil {
		return nil, err
	}
	return &nodeDB{
		lvl:  db,
		self: self,
		quit: make(chan struct{}),
	}, nil
}

func makeKey(id Hash, field string) []byte {
	if bytes.Equal(id[:], nodeDBNilHash[:]) {
		return []byte(field)
	}
	return append(nodeDBItemPrefix, append(id[:], field...)...)
}

func splitKey(key []byte) (id Hash, field string) {
	if !bytes.HasPrefix(key, nodeDBItemPrefix) {
		return Hash{}, string(key)
	}
	item := key[len(nodeDBItemPrefix):]
	copy(id[:], item[:len(id)])
	field = string(item[len(id):])

	return id, field
}

func (db *nodeDB) getNode(id Hash) *Node {
	dbvalue, err := db.lvl.Get(makeKey(id, nodeDBDiscoverRoot), nil)
	if err != nil {
		return nil
	}
	node := &Node{}
	if err := node.Unmarshal(dbvalue); err != nil {
		log.Println("Failed to decode node", "err", err)
		return nil
	}
	return node
}

func (db *nodeDB) updateNode(node *Node) error {
	dbvalue, err := node.Marshal()
	if err != nil {
		return err
	}
	return db.lvl.Put(makeKey(node.GetID(), nodeDBDiscoverRoot), dbvalue, nil)
}

func (db *nodeDB) deleteNode(id Hash) error {
	deleter := db.lvl.NewIterator(util.BytesPrefix(makeKey(id, "")), nil)
	for deleter.Next() {
		if err := db.lvl.Delete(deleter.Key(), nil); err != nil {
			return err
		}
	}
	return nil
}

func (db *nodeDB) getInt64(key []byte) int64 {
	dbvalue, err := db.lvl.Get(key, nil)
	if err != nil {
		return 0
	}
	val, nbyte := binary.Varint(dbvalue)
	if nbyte <= 0 {
		return 0
	}
	return val
}

func (db *nodeDB) storeInt64(key []byte, n int64) error {
	dbvalue := make([]byte, binary.MaxVarintLen64)
	dbvalue = dbvalue[:binary.PutVarint(dbvalue, n)]

	return db.lvl.Put(key, dbvalue, nil)
}

// ensureExpirer is a small helper method ensuring that the data expiration
// mechanism is running. If the expiration goroutine is already running, this
// method simply returns.
//
// The goal is to start the data evacuation only after the network successfully
// bootstrapped itself (to prevent dumping potentially useful seed nodes). Since
// it would require significant overhead to exactly trace the first successful
// convergence, it's simpler to "ensure" the correct state when an appropriate
// condition occurs (i.e. a successful bonding), and discard further events.
func (db *nodeDB) ensureExpirer() {
	db.runner.Do(func() { go db.expirer() })
}

// expirer should be started in a go routine, and is responsible for looping ad
// infinitum and dropping stale data from the database.
func (db *nodeDB) expirer() {
	tick := time.NewTicker(nodeDBCleanupCycle)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if err := db.expireNodes(); err != nil {
				log.Println("Failed to expire nodedb items", "err", err)
			}
		case <-db.quit:
			return
		}
	}
}

func (db *nodeDB) expireNodes() error {
	threshold := time.Now().Add(-nodeDBNodeExpiration)

	// Find discovered nodes that are older than the allowance
	it := db.lvl.NewIterator(nil, nil)

	for it.Next() {
		// Skip the item if not a discovery node
		id, field := splitKey(it.Key())
		if field != nodeDBDiscoverRoot {
			continue
		}
		// Skip the node if not expired yet (and not self)
		if !bytes.Equal(id[:], db.self[:]) {
			if seen := db.lastPongReceived(id); seen.After(threshold) {
				continue
			}
		}
		// Otherwise delete all associated information
		db.deleteNode(id)
	}
	it.Release()
	return nil
}

// lastPingReceived retrieves the time of the last ping packet sent by the remote node.
//not used
func (db *nodeDB) lastPingReceived(id Hash) time.Time {
	return time.Unix(db.getInt64(makeKey(id, nodeDBDiscoverPing)), 0)
}

// updateLastPing updates the last time remote node pinged us.
//not used
func (db *nodeDB) updateLastPingReceived(id Hash, instance time.Time) error {
	return db.storeInt64(makeKey(id, nodeDBDiscoverPing), instance.Unix())
}

// lastPongReceived retrieves the time of the last successful pong from remote node.
func (db *nodeDB) lastPongReceived(id Hash) time.Time {
	return time.Unix(db.getInt64(makeKey(id, nodeDBDiscoverPong)), 0)
}

// hasBond reports whether the given node is considered bonded.
//not used
func (db *nodeDB) hasBond(id Hash) bool {
	return time.Since(db.lastPongReceived(id)) < nodeDBNodeExpiration
}

// updateLastPongReceived updates the last pong time of a node.
func (db *nodeDB) updateLastPongReceived(id Hash, instance time.Time) error {
	return db.storeInt64(makeKey(id, nodeDBDiscoverPong), instance.Unix())
}

// findFails retrieves the number of findnode failures since bonding.
func (db *nodeDB) findFails(id Hash) int {
	return int(db.getInt64(makeKey(id, nodeDBDiscoverFindFails)))
}

// updateFindFails updates the number of findnode failures since bonding.
func (db *nodeDB) updateFindFails(id Hash, fails int) error {
	return db.storeInt64(makeKey(id, nodeDBDiscoverFindFails), int64(fails))
}

// querySeeds retrieves random nodes to be used as potential seed nodes
// for bootstrapping.
func (db *nodeDB) querySeeds(n int, maxAge time.Duration) []*Node {
	var (
		now   = time.Now()
		nodes = make([]*Node, 0, n)
		it    = db.lvl.NewIterator(nil, nil)
		id    Hash
	)
	defer it.Release()

seek:
	for seeks := 0; len(nodes) < n && seeks < n*5; seeks++ {
		// Seek to a random entry. The first byte is incremented by a
		// random amount each time in order to increase the likelihood
		// of hitting all existing nodes in very small databases.
		ctr := id[0]
		rand.Read(id[:])
		id[0] = ctr + id[0]%16
		it.Seek(makeKey(id, nodeDBDiscoverRoot))

		node := nextNode(it)
		if node == nil {
			id[0] = 0
			continue seek // iterator exhausted
		}
		if node.GetID() == db.self {
			continue seek
		}
		if now.Sub(db.lastPongReceived(node.GetID())) > maxAge {
			continue seek
		}
		for i := range nodes {
			if nodes[i].GetID() == node.GetID() {
				continue seek // duplicate
			}
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// reads the next node record from the iterator, skipping over other
// database entries.
func nextNode(it iterator.Iterator) *Node {
	for end := false; !end; end = !it.Next() {
		id, field := splitKey(it.Key())
		if field != nodeDBDiscoverRoot {
			continue
		}
		n := &Node{}
		if err := n.Unmarshal(it.Value()); err != nil {
			log.Println("Failed to decode node", "id", id, "err", err)
			continue
		}
		return n
	}
	return nil
}

// close flushes and closes the database files.
func (db *nodeDB) close() {
	close(db.quit)
	db.lvl.Close()
}
