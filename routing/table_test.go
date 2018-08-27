package routing

import (
	"testing"
	"bytes"
)

var net transport
var tab, _ = NewTable(net,Hash{41},"nc","",[]INode{})

func Test_GetNodeLocally(t *testing.T) {
	//tab.Start()
	n1 := &Node{addr: "na", id: Hash{43}}
	n2 := &Node{addr: "na", id: Hash{44}}
	n3 := &Node{addr: "na", id: Hash{45}}
	n4 := &Node{addr: "nb", id: Hash{46}}
	n5 := &Node{addr: "na", id: Hash{47}}
	n6 := &Node{addr: "na", id: Hash{48}}
	n7 := &Node{addr: "na", id: Hash{49}}
	n8 := &Node{addr: "na", id: Hash{40}}
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
		if buckets[i] != nil{
			tab.add(buckets[i])
		}
	}

	b := tab.GetNodeLocally(Hash{46})
	for _,node := range buckets {
		for _, n := range b {
			if n.GetID() == node.id {
				if n.GetAddr() != node.addr {
					t.Error("something err")
				}
				t.Log("node:",node,"found")
			}
		}
	}


	t.Log("test getnodelocally pass")

}

func Test_closest(t *testing.T) {


	n1 := &Node{addr: "na", id: Hash{43}}
	n2 := &Node{addr: "na", id: Hash{44}}
	n3 := &Node{addr: "na", id: Hash{45}}
	n4 := &Node{addr: "nb", id: Hash{46}}
	n5 := &Node{addr: "na", id: Hash{47}}
	n6 := &Node{addr: "na", id: Hash{48}}
	n7 := &Node{addr: "na", id: Hash{49}}
	n8 := &Node{addr: "na", id: Hash{40}}
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
	if bytes.Equal(nodes.entries[0].id[:], n4.id[:]) {
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
	if distance(a, b) != 156 || distance(a, c) != 148 || distance(a, d) != 153 {
		t.Error("distance wrong")
	} else {
		t.Log("test distance pass")
	}
}

func Test_delete(t *testing.T) {
	n5 := &Node{addr: "na", id: Hash{47}}
	nodes := tab.closest(Hash{47}, 4)
	if bytes.Equal(nodes.entries[0].id[:], n5.id[:]) {
		t.Log("test closest pass again")
	} else {
		t.Error("something wrong")
	}

	tab.delete(n5)
	nodes = tab.closest(Hash{47}, 10)
	if bytes.Equal(nodes.entries[0].id[:], n5.id[:]) {
		t.Error("something wrong")
	} else {
		t.Log("test delete pass")
	}
}