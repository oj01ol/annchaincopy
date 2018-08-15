package routing

import (
	"testing"
)

type node struct{
	addr string
	id	Hash
}

func (n node) GetAddr() string{
	return n.addr
}

func (n node) GetID() Hash{
	return n.id
}

func Test_GetNodeAddress(t *testing.T) {

	var tab table
	tab.selfID = [64]byte{41,}
	n1 := &node{addr:"na",id:[64]byte{43,}}
	n2 := &node{addr:"na",id:[64]byte{44,}}
	n3 := &node{addr:"na",id:[64]byte{45,}}
	n4 := &node{addr:"nb",id:[64]byte{46,}}
	n5 := &node{addr:"na",id:[64]byte{47,}}
	n6 := &node{addr:"na",id:[64]byte{48,}}
	n7 := &node{addr:"na",id:[64]byte{49,}}
	n8 := &node{addr:"na",id:[64]byte{40,}}
	var buckets []*node
	buckets = append(buckets,n1)
	buckets = append(buckets,n2)
	buckets = append(buckets,n3)
	buckets = append(buckets,n4)
	buckets = append(buckets,n5)
	buckets = append(buckets,n6)
	buckets = append(buckets,n7)
	buckets = append(buckets,n8)
	
	for _,m := range buckets{
		tab.bucket = append(tab.bucket,m)
	}
	
	b := tab.GetNodeAddress([64]byte{46,})
	if b[0] == n4 {
		t.Log("test getnodeaddress pass")
	}else {
		t.Error("something wrong")
	}
}

func Test_closest(t *testing.T) {
	var tab table
	tab.selfID = [64]byte{41,}
	n1 := &node{addr:"na",id:[64]byte{43,}}
	n2 := &node{addr:"na",id:[64]byte{44,}}
	n3 := &node{addr:"na",id:[64]byte{45,}}
	n4 := &node{addr:"nb",id:[64]byte{46,}}
	n5 := &node{addr:"na",id:[64]byte{47,}}
	n6 := &node{addr:"na",id:[64]byte{48,}}
	n7 := &node{addr:"na",id:[64]byte{49,}}
	n8 := &node{addr:"na",id:[64]byte{40,}}
	var buckets []*node
	buckets = append(buckets,n1)
	buckets = append(buckets,n2)
	buckets = append(buckets,n3)
	buckets = append(buckets,n4)
	buckets = append(buckets,n5)
	buckets = append(buckets,n6)
	buckets = append(buckets,n7)
	buckets = append(buckets,n8)
	
	for _,m := range buckets{
		tab.bucket = append(tab.bucket,m)
	}
	
	nodes := tab.closest([64]byte{46,}, 4)
	if nodes.entries[0] == n4{
		t.Log("test closest pass")
	}else{
		t.Error("something wrong")
	}
}

func Test_distance(t *testing.T) {
	a := [64]byte{8,8,}
	b := [64]byte{7,255,}
	c := [64]byte{8,7,}
	d := [64]byte{9,255,}
	if distance(a,b) != 508 || distance(a,c) != 500 || distance(a,d) != 505{
		t.Error("distance wrong")
	}else{
		t.Log("test distance pass")
	}
}