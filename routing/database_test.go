package routing

import (
	"io/ioutil"
	"testing"

	"time"

	"github.com/stretchr/testify/require"
)

const (
	tmpDBName = "nodeDB_test"
)

func Test_newNodeDB(t *testing.T) {
	path, err := ioutil.TempDir("", tmpDBName)
	require.Nil(t, err, "make tempdir err")
	self := Hash{}
	db, err := newNodeDB(path, self)
	require.Nil(t, err, "new node db err")
	db.close()

}

func Test_Node(t *testing.T) {
	path, err := ioutil.TempDir("", tmpDBName)
	require.Nil(t, err, "make tempdir err")
	self := Hash{}
	db, err := newNodeDB(path, self)
	require.Nil(t, err, "new node db err")
	defer db.close()
	Id := Hash{45}
	//ti := time.Now()
	node := &Node{Addr: "na", ID: Id, Time: time.Now().Unix()}
	err = db.updateNode(node)
	require.Nil(t, err, "update node err")
	key := makeKey(node.ID, nodeDBDiscoverRoot)
	t.Log("key", key)
	t.Log("node", node)
	nget := db.getNode(node.ID)
	t.Log("nget", nget)

	if *node == *nget {
		t.Log("update node test pass")
	} else {
		t.Error("node wrong")
	}

	err = db.deleteNode(Id)
	if err != nil {
		t.Error(err)
	}
	nget = db.getNode(node.ID)
	t.Log("nget", nget)
	//db.ensureExpirer()
	if nget == nil {
		t.Log("delete node pass")
	} else {
		t.Error("delete node fail: ", nget)
	}

}

func Test_querySeeds(t *testing.T) {
	path := ""
	self := Hash{7, 63, 74}
	db, _ := newNodeDB(path, self)
	defer db.close()
	var node *Node
	var err error
	have := make(map[Hash]struct{})
	want := make(map[Hash]struct{})
	node = &Node{Addr: "na", ID: Hash{7, 63, 74}, Time: time.Now().Unix()}
	err = db.updateNode(node)
	if err != nil {
		t.Error(err)
	}
	//want[node.ID] = struct{}{}
	node = &Node{Addr: "nb", ID: Hash{4, 26, 84}, Time: time.Now().Unix()}
	err = db.updateNode(node)
	if err != nil {
		t.Error(err)
	}
	want[node.ID] = struct{}{}
	node = &Node{Addr: "nc", ID: Hash{10, 14, 24}, Time: time.Now().Unix()}
	err = db.updateNode(node)
	if err != nil {
		t.Error(err)
	}
	want[node.ID] = struct{}{}
	err = db.updateLastPongReceived(Hash{7, 63, 74}, time.Now())
	err = db.updateLastPongReceived(Hash{4, 26, 84}, time.Now())
	err = db.updateLastPongReceived(Hash{10, 14, 24}, time.Now())
	nodes := db.querySeeds(4, time.Hour*12)
	for _, node = range nodes {
		have[node.ID] = struct{}{}
	}
	if len(have) != len(want) {
		t.Error("quert count mistake")
	}

	for id := range have {
		if _, ok := want[id]; !ok {
			t.Error("extra missed : ", id)
		}
	}

}
