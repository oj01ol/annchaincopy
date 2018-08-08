package demo

import (
	"testing"
)




	
func Test_GetNodeAddress(t *testing.T) {
	var tab table

	var buckets []*Node
	tab.selfID = 41
	n1 := &Node{address:"na",ID:43}
	n2 := &Node{address:"na",ID:44}
	n3 := &Node{address:"na",ID:45}
	n4 := &Node{address:"nb",ID:46}
	n5 := &Node{address:"na",ID:47}
	n6 := &Node{address:"na",ID:48}
	n7 := &Node{address:"na",ID:49}
	n8 := &Node{address:"na",ID:40}
	buckets = append(buckets,n1)
	buckets = append(buckets,n2)
	buckets = append(buckets,n3)
	buckets = append(buckets,n4)
	buckets = append(buckets,n5)
	buckets = append(buckets,n6)
	buckets = append(buckets,n7)
	buckets = append(buckets,n8)
	tab.bucket = buckets
	buckets = tab.GetNodeAddress(42)
	if buckets[0].ID == 42 && buckets[0].address == "ha" {
		buckets = tab.GetNodeAddress(46)
		if buckets[0].ID == 46 && buckets[0].address == "nb"{
			t.Log("test getnodeaddress pass")
		}else{
			t.Error("something wrong")
		}
	}else {
		t.Error("something wrong")
	}
}

func Test_closest(t *testing.T) {
	var tab table

	var buckets []*Node
	tab.selfID = 41
	n1 := &Node{address:"na",ID:43}
	n2 := &Node{address:"na",ID:44}
	n3 := &Node{address:"na",ID:45}
	n4 := &Node{address:"nb",ID:46}
	n5 := &Node{address:"na",ID:47}
	n6 := &Node{address:"na",ID:48}
	n7 := &Node{address:"na",ID:49}
	n8 := &Node{address:"na",ID:40}
	buckets = append(buckets,n1)
	buckets = append(buckets,n2)
	buckets = append(buckets,n3)
	buckets = append(buckets,n4)
	buckets = append(buckets,n5)
	buckets = append(buckets,n6)
	buckets = append(buckets,n7)
	buckets = append(buckets,n8)
	tab.bucket = buckets
	nodes := tab.closest(46,4)
	if nodes.entries[0] == n4{
		t.Log("test closest pass")
	}else{
		t.Error("something wrong")
	}
}

