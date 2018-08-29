package routing

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	ERR_TEST_NODE_NOT_FIND = errors.New("test node not find")
	//, _        =
	TEST_SELF_ID   = Hash{}
	TEST_SELF_ADDR = "127.0.0.1:666"
	TEST_DB_NAME   = "lvdb_test"
)

func init() {
	hash, _ := hex.DecodeString("bc977d652d1853e114ee69bfed4fdaa039149820")
	copy(TEST_SELF_ID[:], hash)
}

var net transport
var tab, _ = NewTable(net, Hash{41}, "nc", "", []INode{})

type TransferForTest struct {
	transferMap map[string]*Table // map[addr]*Table
}

func NewTransferForTest() *TransferForTest {
	transmap := make(map[string]*Table)
	transmap["nc"] = tab
	return &TransferForTest{
		transferMap: transmap,
	}
}

func (tf *TransferForTest) Ping(addr string) error {
	if _, ok := tf.transferMap[addr]; ok {
		return nil
	}
	return ERR_TEST_NODE_NOT_FIND
}

func (tf *TransferForTest) FindNode(addr string, target Hash) ([]INode, error) {
	if t, ok := tf.transferMap[addr]; ok {
		return t.GetNodeLocally(target), nil
	}
	return nil, ERR_TEST_NODE_NOT_FIND
}

func randHashForTest() (ret Hash) {
	crand.Read(ret[:])
	return
}

func genBootNodes(num int) []INode {
	infos := make([]INode, num)
	for i := range infos {
		node := &Node{}
		node.Addr = fmt.Sprintf("10.10.10.%v:%v", rand.Intn(255), rand.Intn(25))
		node.ID = randHashForTest()
		infos[i] = node
		//fmt.Printf("gen nodes:%x,%v\n", infos[i].GetID(), infos[i].GetAddr())
	}
	return infos
}

func TestNewTable(t *testing.T) {
	tsfer := NewTransferForTest()
	dbpath, err := ioutil.TempDir("", TEST_DB_NAME)
	require.Nil(t, err, "get temp dir err")
	tb, err := NewTable(tsfer, TEST_SELF_ID, TEST_SELF_ADDR, dbpath, genBootNodes(10))
	require.Nil(t, err, "new table err")
	tb.Start()
	tb.Stop()
}

func Test_GetNodeLocally(t *testing.T) {
	//tab.Start()
	n1 := &Node{Addr: "na", ID: Hash{43}}
	n2 := &Node{Addr: "na", ID: Hash{44}}
	n3 := &Node{Addr: "na", ID: Hash{45}}
	n4 := &Node{Addr: "nb", ID: Hash{46}}
	n5 := &Node{Addr: "na", ID: Hash{47}}
	n6 := &Node{Addr: "na", ID: Hash{48}}
	n7 := &Node{Addr: "na", ID: Hash{49}}
	n8 := &Node{Addr: "na", ID: Hash{40}}
	var buckets []*Node
	buckets = append(buckets, n1)
	buckets = append(buckets, n2)
	buckets = append(buckets, n3)
	buckets = append(buckets, n4)
	buckets = append(buckets, n5)
	buckets = append(buckets, n6)
	buckets = append(buckets, n7)
	buckets = append(buckets, n8)

	for i := range buckets {
		if buckets[i] != nil {
			tab.add(buckets[i])
		}
	}

	b := tab.GetNodeLocally(Hash{46})
	for _, node := range buckets {
		for _, n := range b {
			if n.GetID() == node.GetID() {
				if n.GetAddr() != node.Addr {
					t.Error("something err")
				}
				t.Log("node:", node, "found")
			}
		}
	}

	t.Log("test getnodelocally pass")

}

func Test_closest(t *testing.T) {

	n1 := &Node{Addr: "na", ID: Hash{43}}
	n2 := &Node{Addr: "na", ID: Hash{44}}
	n3 := &Node{Addr: "na", ID: Hash{45}}
	n4 := &Node{Addr: "nb", ID: Hash{46}}
	n5 := &Node{Addr: "na", ID: Hash{47}}
	n6 := &Node{Addr: "na", ID: Hash{48}}
	n7 := &Node{Addr: "na", ID: Hash{49}}
	n8 := &Node{Addr: "na", ID: Hash{40}}
	var buckets []*Node
	buckets = append(buckets, n1)
	buckets = append(buckets, n2)
	buckets = append(buckets, n3)
	buckets = append(buckets, n4)
	buckets = append(buckets, n5)
	buckets = append(buckets, n6)
	buckets = append(buckets, n7)
	buckets = append(buckets, n8)

	for _, m := range buckets {
		tab.add(m)
	}

	nodes := tab.closest(Hash{46}, 4)
	if bytes.Equal(nodes.entries[0].ID[:], n4.ID[:]) {
		t.Log("test closest pass")
	} else {
		t.Error("something wrong")
	}
}

func Test_distance(t *testing.T) {
	a := Hash{8, 8}
	b := Hash{7, 255}
	c := Hash{8, 7}
	d := Hash{9, 255}
	if distance(a, b) != 316 || distance(a, c) != 308 || distance(a, d) != 313 {
		t.Error("distance wrong")
	} else {
		t.Log("test distance pass")
	}
}

func Test_delete(t *testing.T) {

	n5 := &Node{Addr: "na", ID: Hash{47}}
	nodes := tab.closest(Hash{47}, 4)
	if bytes.Equal(nodes.entries[0].ID[:], n5.ID[:]) {
		t.Log("test closest pass again")
	} else {
		t.Error("something wrong")
	}

	tab.delete(n5)
	nodes = tab.closest(Hash{47}, 10)

	if bytes.Equal(nodes.entries[0].ID[:], n5.ID[:]) {
		t.Error("something wrong")
	} else {
		t.Log("test delete pass")
	}
}

func Test_GetNodeByNet(t *testing.T) {
	tsfer := NewTransferForTest()
	dbpath, err := ioutil.TempDir("", TEST_DB_NAME)
	require.Nil(t, err, "get temp dir err")
	tb, err := NewTable(tsfer, TEST_SELF_ID, TEST_SELF_ADDR, dbpath, genBootNodes(10))
	require.Nil(t, err, "new table err")
	tb.Start()
	defer tb.Stop()
	ntab := &Node{Addr: "nc", ID: Hash{41}}
	tb.add(ntab)
	Id := Hash{46}
	nodes := tb.GetNodeByNet(Id)
	for _, node := range nodes {
		if node.GetID() == Id {
			if node.GetAddr() == "nb" {
				t.Log("test GetNodeByNet pass")
				return
			}
		}
	}
	t.Error("not found by net")
}
