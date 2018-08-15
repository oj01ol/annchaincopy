package routing

import (
	"bytes"
	"encoding/json"
	"log"
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"
	
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/syndtr/goleveldb/leveldb/storage"

)

var (
	nodeDBNilHash      = Hash{}       // Special node ID to use as a nil element.
	nodeDBNodeExpiration = 24 * time.Hour // Time after which an unseen node should be dropped.
	nodeDBCleanupCycle   = time.Hour      // Time period for running the expiration task.
	
	nodeDBItemPrefix = []byte("n:")      // Identifier to prefix node entries

	nodeDBDiscoverRoot      = ":discover"
	nodeDBDiscoverPing      = nodeDBDiscoverRoot + ":lastping"
	nodeDBDiscoverPong      = nodeDBDiscoverRoot + ":lastpong"
	nodeDBDiscoverFindFails = nodeDBDiscoverRoot + ":findfail"
)

type nodeDB struct {
	lvl		*leveldb.DB
	self	Hash
	runner 	sync.Once     // Ensures we can start at most one expirer
	quit	chan struct{}
}

func newNodeDB(path string, self Hash) (*nodeDB, error){
	var db *leveldb.DB
	var err error
	if path == "" {
		db, err = leveldb.Open(storage.NewMemStorage(), nil)
	}else {
		db, err = leveldb.OpenFile(path, nil)
	}

	if err != nil{
		return nil,err
	}
	return &nodeDB{
		lvl:	db,
		self:	self,
		quit:	make(chan struct{}),
	}, nil
}


func makeKey(id Hash, field string) []byte {
	if bytes.Equal(id[:], nodeDBNilHash[:]){
		return []byte(field)
	}
	return append(nodeDBItemPrefix,append(id[:],field...)...)
}

func splitKey(key []byte) (id Hash, field string) {
	if !bytes.HasPrefix(key, nodeDBItemPrefix){
		return Hash{},string(key)
	}
	item := key[len(nodeDBItemPrefix):]
	copy(id[:],item[:len(id)])
	field = string(item[len(id):])
	
	return id , field
}

func (db *nodeDB) getNode(id Hash) *tNode {
	dbvalue, err := db.lvl.Get(makeKey(id, nodeDBDiscoverRoot), nil)
	if err != nil {
		return nil
	}
	var node tNode
	if err := json.Unmarshal(dbvalue, &node); err != nil {
		log.Println("Failed to decode node json", "err", err)
		return nil
	}
	return &node
} 


func (db *nodeDB) updateNode(node *tNode) error{
	dbvalue, err := json.Marshal(node)
	if err != nil {
		return err
	}
	return db.lvl.Put(makeKey(node.ID, nodeDBDiscoverRoot), dbvalue, nil)
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
	defer it.Release()

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
	return nil
}


// lastPingReceived retrieves the time of the last ping packet sent by the remote node.
func (db *nodeDB) lastPingReceived(id Hash) time.Time {
	return time.Unix(db.getInt64(makeKey(id, nodeDBDiscoverPing)), 0)
}

// updateLastPing updates the last time remote node pinged us.
func (db *nodeDB) updateLastPingReceived(id Hash, instance time.Time) error {
	return db.storeInt64(makeKey(id, nodeDBDiscoverPing), instance.Unix())
}

// lastPongReceived retrieves the time of the last successful pong from remote node.
func (db *nodeDB) lastPongReceived(id Hash) time.Time {
	return time.Unix(db.getInt64(makeKey(id, nodeDBDiscoverPong)), 0)
}

// hasBond reports whether the given node is considered bonded.
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
func (db *nodeDB) querySeeds(n int, maxAge time.Duration) []*tNode {
	var (
		now   = time.Now()
		nodes = make([]*tNode, 0, n)
		it    = db.lvl.NewIterator(nil, nil)
		Id    Hash
	)
	defer it.Release()

seek:
	for seeks := 0; len(nodes) < n && seeks < n*5; seeks++ {
		// Seek to a random entry. The first byte is incremented by a
		// random amount each time in order to increase the likelihood
		// of hitting all existing nodes in very small databases.
		ctr := Id[0]
		rand.Read(Id[:])
		Id[0] = ctr + Id[0]%16
		it.Seek(makeKey(Id, nodeDBDiscoverRoot))

		n := nextNode(it)
		if n == nil {
			Id[0] = 0
			continue seek // iterator exhausted
		}
		if n.ID == db.self {
			continue seek
		}
		if now.Sub(db.lastPongReceived(n.ID)) > maxAge {
			continue seek
		}
		for i := range nodes {
			if nodes[i].ID == n.ID {
				continue seek // duplicate
			}
		}
		nodes = append(nodes, n)
	}
	return nodes
}

// reads the next node record from the iterator, skipping over other
// database entries.
func nextNode(it iterator.Iterator) *tNode {
	for end := false; !end; end = !it.Next() {
		id, field := splitKey(it.Key())
		if field != nodeDBDiscoverRoot {
			continue
		}
		var n tNode
		if err := json.Unmarshal(it.Value(), &n); err != nil {
			log.Println("Failed to decode node json", "id", id, "err", err)
			continue
		}
		return &n
	}
	return nil
}

// close flushes and closes the database files.
func (db *nodeDB) close() {
	close(db.quit)
	db.lvl.Close()
}

