package routing

import (
	"bytes"
	"testing"
)

func Test_GetNodeAddress(t *testing.T) {
	var tab Table
	tab.self.id = Hash{41}
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
		tab.add(buckets[i])
	}

	b := tab.GetNodeAddress(Hash{46})
	if b == n4.addr {
		t.Log("test getnodeaddress pass")
	} else {
		t.Error("something wrong")
	}
}

func Test_closest(t *testing.T) {
	var tab Table
	tab.self.id = Hash{41}
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
	if distance(a, b) != 508 || distance(a, c) != 500 || distance(a, d) != 505 {
		t.Error("distance wrong")
	} else {
		t.Log("test distance pass")
	}
}
