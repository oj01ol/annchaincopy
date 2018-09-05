package routing

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	ERR_TEST_NODE_NOT_FIND = errors.New("test node not find")
	//, _        =
	TEST_SELF_ID   Hash
	TEST_SELF_ADDR = "127.0.0.1:666"
	TEST_DB_NAME   = "lvdb_test"
	net            transport
	inited         bool
	TMP_DIR        = "./tmp_dir_for_test"
)

func initTest() {
	if inited {
		return
	}
	inited = true

	initConfig()
	hash, _ := hex.DecodeString("bc977d652d1853e114ee69bfed4fdaa039149820")
	TEST_SELF_ID = ToHash(hash)
}

type TransferForTest struct {
	transferMap sync.Map //// map[addr]*Table
	tmpPath     []string
}

func (tf *TransferForTest) Clear() {
	for i := range tf.tmpPath {
		os.Remove(tf.tmpPath[i])
	}
}

func (tf *TransferForTest) fillSpecificData(t *testing.T, nodes []INode, initNum int) {
	tf.tmpPath = make([]string, len(nodes))
	for i := range nodes {
		tbID := nodes[i].GetID()
		tbIP := nodes[i].GetAddr()
		dbpath, err := ioutil.TempDir("", fmt.Sprintf("tem_db_%v", i+1))
		assert.Nil(t, err, "new dbpath err")
		tf.tmpPath[i] = dbpath
		choose := chooseFromNodes(nodes, initNum, i)
		tb, err := NewTable(tf, tbID, tbIP, dbpath, choose)
		choosedN := _inodesToNodes(choose)
		for j := range choosedN {
			tb.add(choosedN[j])
		}
		assert.Nil(t, err, "new table err")
		tf.transferMap.Store(tbIP, tb)
	}
}

func (tf *TransferForTest) ExecAll(exec func(t *Table) bool) {
	tf.transferMap.Range(func(key, value interface{}) bool {
		if !exec(value.(*Table)) {
			return false
		}
		return true
	})
}

func (tf *TransferForTest) Start(start bool) {
	tf.ExecAll(func(t *Table) bool {
		if start {
			t.Start()
		} else {
			t.Stop()
		}
		return true
	})
}

func (tf *TransferForTest) Ping(addr string) error {
	if _, ok := tf.transferMap.Load(addr); ok {
		return nil
	}
	return ERR_TEST_NODE_NOT_FIND
}

func (tf *TransferForTest) FindNode(addr string, target Hash) ([]INode, error) {
	if t, ok := tf.transferMap.Load(addr); ok {
		return t.(*Table).GetNodeLocally(target), nil
	}
	return nil, ERR_TEST_NODE_NOT_FIND
}

func genIPForTest(id int) string {
	return fmt.Sprintf("123.123.123.%v:%v", id, id)
}

func randHashForTest() (ret Hash) {
	ret = NewHash()
	crand.Read(ret)
	return
}

func genBootNodes(num int) []INode {
	infos := make([]INode, num)
	for i := range infos {
		node := &Node{}
		node.Addr = genIPForTest(i + 1)
		node.ID = randHashForTest()
		infos[i] = node
		//fmt.Printf("gen nodes:%x,%v\n", infos[i].GetID(), infos[i].GetAddr())
	}
	return infos
}

func chooseFromNodes(nodes []INode, num, except int) []INode {
	cpNodes := make([]INode, len(nodes))
	copy(cpNodes, nodes)
	from := cpNodes[:except]
	if except < len(cpNodes)-1 {
		from = append(from, cpNodes[except+1:len(cpNodes)]...)
	}
	if num >= len(from) {
		return from
	}
	if num == 1 {
		choose := []INode{from[except%len(from)]}
		return choose
	}
	rand.Seed(time.Now().Unix() + int64(num))
	res := rand.Perm(len(from))
	choose := make([]INode, num)
	for i := range res {
		choose[i] = from[res[i]]
	}
	return choose
}

func TestNewTable(t *testing.T) {
	initTest()
	nodes := genBootNodes(3)
	tsfer := &TransferForTest{}
	dbpath, err := ioutil.TempDir("", TEST_DB_NAME)
	defer os.Remove(dbpath)
	require.Nil(t, err, "get temp dir err")
	tb, err := NewTable(tsfer, TEST_SELF_ID, TEST_SELF_ADDR, dbpath, nodes)
	require.Nil(t, err, "new table err")
	tb.Start()
	tb.Stop()
}

func Test_GetNodeLocally(t *testing.T) {
	initTest()
	tab, _ := NewTable(net, ToHash([]byte{41}), "nc", "", []INode{})
	n1 := &Node{Addr: "na", ID: ToHash([]byte{43})}
	n2 := &Node{Addr: "na", ID: ToHash([]byte{44})}
	n3 := &Node{Addr: "na", ID: ToHash([]byte{45})}
	n4 := &Node{Addr: "nb", ID: ToHash([]byte{46})}
	n5 := &Node{Addr: "na", ID: ToHash([]byte{47})}
	n6 := &Node{Addr: "na", ID: ToHash([]byte{48})}
	n7 := &Node{Addr: "na", ID: ToHash([]byte{49})}
	n8 := &Node{Addr: "na", ID: ToHash([]byte{40})}
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
		tab.add(buckets[i])
	}

	b := tab.GetNodeLocally(ToHash([]byte{46}))
	for _, node := range buckets {
		for _, n := range b {
			if n.GetID().Equal(node.GetID()) {
				if n.GetAddr() != node.Addr {
					t.Errorf("addr not equal,id:%v,ori:%v,get:%v", n.GetID(), node.Addr, n.GetAddr())
				}
			}

		}
	}
}

func Test_closest(t *testing.T) {
	initTest()
	tab, _ := NewTable(net, ToHash([]byte{41}), "nc", "", []INode{})
	n1 := &Node{Addr: "na", ID: ToHash([]byte{43})}
	n2 := &Node{Addr: "na", ID: ToHash([]byte{44})}
	n3 := &Node{Addr: "na", ID: ToHash([]byte{45})}
	n4 := &Node{Addr: "nb", ID: ToHash([]byte{46})}
	n5 := &Node{Addr: "na", ID: ToHash([]byte{47})}
	n6 := &Node{Addr: "na", ID: ToHash([]byte{48})}
	n7 := &Node{Addr: "na", ID: ToHash([]byte{49})}
	n8 := &Node{Addr: "na", ID: ToHash([]byte{40})}
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

	nodes := tab.closest(ToHash([]byte{46}), 4)
	if !bytes.Equal(nodes.entries[0].ID[:], n4.ID[:]) {
		t.Errorf("wrong nodes closest id, get:%x,expected:%x", nodes.entries[0].ID[:], n4.ID[:])
	}
}

func Test_distance(t *testing.T) {
	initTest()
	a := ToHash([]byte{8, 8})
	b := ToHash([]byte{7, 255})
	c := ToHash([]byte{8, 7})
	d := ToHash([]byte{9, 255})
	if distance(a, b) != 316 || distance(a, c) != 308 || distance(a, d) != 313 {
		t.Error("distance wrong")
	}
}

func Test_delete(t *testing.T) {
	initTest()
	n5 := &Node{Addr: "na", ID: ToHash([]byte{47})}
	tab, _ := NewTable(net, ToHash([]byte{41}), "nc", "", []INode{})
	tab.add(n5)
	nodes := tab.closest(ToHash([]byte{47}), 4)
	if !nodes.entries[0].ID.Equal(n5.ID) {
		t.Errorf("wrong nodes closest id, get:%x,expected:%x", nodes.entries[0].ID[:], n5.ID[:])
	}

	tab.delete(n5)
	nodes = tab.closest(ToHash([]byte{47}), 10)

	if len(nodes.entries) > 0 && nodes.entries[0].ID.Equal(n5.ID) {
		t.Error("not deleted")
	}
}

func Test_GetNodeByNet(t *testing.T) {
	initTest()
	tsfer := &TransferForTest{}
	dbpath, err := ioutil.TempDir("", TEST_DB_NAME)
	assert.Nil(t, err, "get temp dir err")
	defer os.Remove(dbpath)

	allNodes := genBootNodes(3)

	// num 1 is special for the test net
	tsfer.fillSpecificData(t, allNodes, 1)
	tsfer.Start(true)
	defer func() {
		tsfer.Start(false)
		tsfer.Clear()
	}()

	for i := range allNodes {
		local := allNodes[i]
		var find bool
		tsfer.ExecAll(func(remote *Table) bool {
			if remote.self.ID.Equal(local.GetID()) {
				return true
			}
			// stored in other node's table
			if addr := remote.GetNodeAddr(local.GetID()); addr == local.GetAddr() {
				find = true
			}
			return !find
		})
		if !find {
			t.Errorf("can't get addr of nodes[%x],real addr:%v.\n", local.GetID(), local.GetAddr())
		}
	}
}
