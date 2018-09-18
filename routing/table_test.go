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
	TEST_SELF_ID           Hash
	TEST_SELF_ADDR         = "127.0.0.1:666"
	TEST_DB_NAME           = "lvdb_test"
	net                    transport
	TMP_DIR                = "./tmp_dir_for_test"
)

func initTest() {
	Init(NewConfigurable())
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
		choose := chooseFromNodes(nodes /*[:i+1]*/, initNum, i)
		tb, err := NewTable(tf, tbID, tbIP, dbpath, choose)
		choosedN := _inodesToNodes(choose)
		for j := range choosedN {
			tb.add(choosedN[j])
		}
		assert.Nil(t, err, "new table err")
		tf.transferMap.Store(tbIP, tb)
	}
}

func (tf *TransferForTest) findNodeForcely(t *testing.T) {
	tf.ExecAll(func(tb *Table) bool {

		targetID := tb.self.GetID()
		result := tb.closest(targetID, c.findsize)
		seen := make(map[HashKey]bool)

		for i := 0; i < len(result.entries); i++ {
			fromNode := result.entries[i]
			reply := make(chan []*Node, c.alpha+1)
			tb.findNode(fromNode, targetID, reply)
			tf.onReq(fromNode.Addr, tb.self)
			for i := 0; i < c.alpha; i++ {
				targetID = NewHash()
				crand.Read(targetID[:])
				tb.findNode(fromNode, targetID, reply)
				tf.onReq(fromNode.Addr, tb.self)
			}
			for _, retNode := range <-reply {
				//fmt.Println("======from====", fromNode.Addr, retNode.Addr)
				if retNode != nil {
					nodeKey := retNode.GetID().AsKey()
					if !seen[nodeKey] {
						seen[nodeKey] = true
						result.push(retNode, c.findsize)
					}
				}
			}
		}

		return true
	})
}

func (tf *TransferForTest) onReq(addr string, reqN INode) {
	if t, ok := tf.transferMap.Load(addr); ok {
		t.(*Table).OnReceiveReq(reqN)
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
		tbb := t.(*Table)
		return tbb.GetNodeLocally(target), nil
	}
	return nil, ERR_TEST_NODE_NOT_FIND
}

func genIPForTest(id int) string {
	return fmt.Sprintf("123.123.123.%v:%v", id, id)
}

func randHashForTest() (ret Hash) {
	ret = NewHash()
	crand.Read(ret)
	//ret[0] = 'a'
	//	ret[1] = 'a'
	//	ret[2] = 'a'
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

// chooseFromNodes choose randum num nodes in param 'nodes'
// except nodes index except
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
	rand.Seed(time.Now().UnixNano())
	res := rand.Perm(len(from))
	choose := make([]INode, num)
	for i := range res[:num] {
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

	// A{B},B{C},C{A},test whether A,B,C can connect to each other
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

func Test_BrainSplitRate(t *testing.T) {
	initTest()
	tsfer := &TransferForTest{}
	dbpath, err := ioutil.TempDir("", TEST_DB_NAME)
	assert.Nil(t, err, "get temp dir err")
	defer os.Remove(dbpath)

	var (
		NODES_NUM         = 500
		INITIAL_NODES_NUM = 3
		TEST_NUM          = 5
	)
	defer func() {
		tsfer.Start(false)
	}()

	t.Logf("\nall num:%v,initial num:%v.\n",
		NODES_NUM, INITIAL_NODES_NUM)

	for i := 0; i < TEST_NUM; i++ {
		allNodes := genBootNodes(NODES_NUM)

		tsfer.fillSpecificData(t, allNodes, INITIAL_NODES_NUM)
		tsfer.findNodeForcely(t)
		tsfer.Start(true)

		fmt.Println()

		//	tsfer.ExecAll(func(t *Table) bool {
		//		fmt.Println("=====", t.self.Addr, t.buketsCount())
		//		return true
		//	})

		fmt.Println()

		var canFind bool
		var notFound, found int
		cantFind := make(map[string][]string) // map[from node][](cant find node)
		missNode := make(map[string]bool)     // map[splite node]

		for i := range allNodes {
			local := allNodes[i]
			canFind = false
			tsfer.ExecAll(func(remote *Table) bool {
				if remote.self.ID.Equal(local.GetID()) {
					return true
				}
				// stored in other node's table
				if addr := remote.GetNodeAddr(local.GetID()); addr == local.GetAddr() {
					found++
					canFind = true
				} else {
					if v, ok := cantFind[remote.self.Addr]; ok {
						cantFind[remote.self.Addr] = append(v, local.GetAddr())
					} else {
						cantFind[remote.self.Addr] = []string{local.GetAddr()}
					}
					notFound++
				}
				return true
			})
			if !canFind {
				missNode[local.GetAddr()] = true
			}
		}
		//	for k, v := range cantFind {
		//		fmt.Printf("from node:%v,cant find:%v\n", k, v)
		//	}
		for k, _ := range missNode {
			fmt.Println("miss node:", k)
		}
		t.Logf("\nnot found num:%v,found num:%v,rate:%v\n",
			notFound, found, float32(notFound)/float32(notFound+found)*100)
		tsfer.Clear()
		fmt.Println("==============================")
	}
}
