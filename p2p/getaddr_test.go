package p2p

import (
	"testing"
)

type node struct{
	addr	string
	id	Hash
}

func (n node) address() string{
	return n.addr
}

func (n node) ID() Hash{
	return n.id
}

func Test_GetNodeAddress(t *testing.T) {

	var buckets []Node
	n1 := node{addr:"na",id:43}
	n2 := node{addr:"na",id:44}
	n3 := node{addr:"na",id:45}
	n4 := node{addr:"nb",id:46}
	n5 := node{addr:"na",id:47}
	n6 := node{addr:"na",id:48}
	n7 := node{addr:"na",id:49}
	n8 := node{addr:"na",id:40}
	buckets = append(buckets,n1)
	buckets = append(buckets,n2)
	buckets = append(buckets,n3)
	buckets = append(buckets,n4)
	buckets = append(buckets,n5)
	buckets = append(buckets,n6)
	buckets = append(buckets,n7)
	buckets = append(buckets,n8)
	buckets = GetNodeAddress(buckets, 46)
	if buckets[0].ID() == 46 && buckets[0].address() == "nb" {
		t.Log("test getnodeaddress pass")
	}else {
		t.Error("something wrong")
	}
}

func Test_closest(t *testing.T) {
	var buckets []Node
	n1 := node{addr:"na",id:43}
	n2 := node{addr:"na",id:44}
	n3 := node{addr:"na",id:45}
	n4 := node{addr:"nb",id:46}
	n5 := node{addr:"na",id:47}
	n6 := node{addr:"na",id:48}
	n7 := node{addr:"na",id:49}
	n8 := node{addr:"na",id:40}
	buckets = append(buckets,n1)
	buckets = append(buckets,n2)
	buckets = append(buckets,n3)
	buckets = append(buckets,n4)
	buckets = append(buckets,n5)
	buckets = append(buckets,n6)
	buckets = append(buckets,n7)
	buckets = append(buckets,n8)
	
	nodes := closest(buckets, 46, 4)
	if nodes.entries[0] == n4{
		t.Log("test closest pass")
	}else{
		t.Error("something wrong")
	}
}