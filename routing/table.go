package routing

import (
	//"fmt"
	"sync"
	//"net"
	//"math/rand"
	//"log"
)

const (
	alpha = 3
	findsize = 16
)

type table struct {
	bucket	[]Node
	mutex	sync.Mutex
	selfID	Hash
}

type Hash [64]byte

type Node interface {
	GetAddr()	string
	GetID()	Hash
}

type nodesByDistance struct {
	entries []Node
	target	Hash
}


func Newtable() (*table , error){
	return nil,nil
}

//get node address by Nodeid
//1.find in the bucket
//2.findnode in the network
func (t *table)	GetNodeAddress(targetID Hash) []Node {
	var (
		asked	= make(map[Hash]bool)
		result	*nodesByDistance
		seen	= make(map[Hash]bool)
		reply	= make(chan []Node, alpha)
		pendingQueries = 0
	)
	
	asked[t.selfID] = true
	
	t.mutex.Lock()
	result = t.closest(targetID, findsize)
	t.mutex.Unlock()
	
	for _,node := range result.entries {
		if node.GetID() == targetID{
			return result.entries
		}
	}
	
	
	for {
		for i := 0; i < len(result.entries) && pendingQueries < alpha; i++ {
			n := result.entries[i]
			if !asked[n.GetID()] {
				asked[n.GetID()] = true
				//seen[n.ID] = true
				pendingQueries++
				
				go t.findnode(n, targetID, reply)
			}
		}
		if pendingQueries == 0 {
			// we have asked all closest nodes, stop the search
			break
		}
		// wait for the next reply
		for _, n := range <-reply {
			if n != nil && !seen[n.GetID()] {
				seen[n.GetID()] = true
				result.push(n, findsize)
			}
		}
		pendingQueries--
	}
	
	return result.entries
	
}

//ask n for the Node info
func (t *table) findnode(n Node, targetID Hash, reply chan<- []Node) {
	
	//send and receive simulation
	nodes := make([]Node, 0, findsize)
	/*
	for i := 0; i < findsize; i++ {
		x:= rand.Intn(100)
		node := &Node{address:"ha",ID:Hash(x)}
		nodes = append(nodes , node)
	}
	*/
	reply <- nodes
}

func (t *table) closest(target Hash, nresults int) *nodesByDistance {
	closeset := &nodesByDistance{target: target}
	for _, n := range t.bucket {
		if n != nil {
			closeset.push(n, nresults)
		}
	}
	return closeset
}

func (h *nodesByDistance) push(n Node, maxElems int) {
	h.entries = append(h.entries,n)
	for i , node := range h.entries{
		if distance(node.GetID() , h.target) > distance(n.GetID() , h.target) {
			copy(h.entries[i+1:],h.entries[i:])
			h.entries[i] = n
			break
		}
	}
	if len(h.entries) > maxElems {
		h.entries = h.entries[:maxElems]
	}
}

func distance(a Hash, b Hash) int {
	lz := 0
	for i := range a {
		x := a[i] ^ b[i]
		if x == 0 {
			lz += 8
		} else {
			lz += lzcount[x]
			break
		}
	}
	return len(a)*8 - lz
	
}

var lzcount = [256]int{
	8, 7, 6, 6, 5, 5, 5, 5,
	4, 4, 4, 4, 4, 4, 4, 4,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
}

