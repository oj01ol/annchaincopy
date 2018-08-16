package routing

import (
	"testing"
	
	"time"
)

func Test_newNodeDB(t *testing.T) {
	path := "./nodeDB"
	self := Hash{}
	db,err := newNodeDB(path,self)
	t.Log(err)
	db.close()
	
}

func Test_Node(t *testing.T) {
	path := "./nodeDB"
	self := Hash{}
	db,_ := newNodeDB(path,self)
	defer db.close()
	Id := [20]byte{45,}
	//ti := time.Now()
	node := &tNode{Addr:"na",ID:Id,AddTime:time.Now().Unix()}
	err := db.updateNode(node)
	if err != nil {
		t.Error(err)
	}
	key:=makeKey(node.ID, nodeDBDiscoverRoot)
	t.Log("key",key)
	t.Log("node",node)
	nget := db.getNode(node.ID)
	t.Log("nget", nget)
	
	
	if *node == *nget {
		t.Log("update node test pass")
	}else{
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
	}else {
		t.Error("delete node fail: ",nget)
	}
	
	
}

func Test_querySeeds(t *testing.T){
	path := ""
	self := [20]byte{7,63,74}
	db,_ := newNodeDB(path,self)
	defer db.close()
	var node *tNode
	var err error
	have := make(map[Hash]struct{})
	want := make(map[Hash]struct{})
	node = &tNode{Addr:"na",ID:[20]byte{7,63,74},AddTime:time.Now().Unix()}
	err = db.updateNode(node)
	if err != nil {
		t.Error(err)
	}
	//want[node.ID] = struct{}{}
	node = &tNode{Addr:"nb",ID:[20]byte{4,26,84},AddTime:time.Now().Unix()}
	err = db.updateNode(node)
	if err != nil {
		t.Error(err)
	}
	want[node.ID] = struct{}{}
	node = &tNode{Addr:"nc",ID:[20]byte{10,14,24},AddTime:time.Now().Unix()}
	err = db.updateNode(node)
	if err != nil {
		t.Error(err)
	}
	want[node.ID] = struct{}{}
	err = db.updateLastPongReceived([20]byte{7,63,74},time.Now())
	err = db.updateLastPongReceived([20]byte{4,26,84},time.Now())
	err = db.updateLastPongReceived([20]byte{10,14,24},time.Now())
	nodes := db.querySeeds(4,time.Hour*12)
	for _,node = range nodes{
		have[node.ID] = struct{}{}
	}
	if len(have) != len(want){
		t.Error("quert count mistake")
	}
	
	for id := range have {
		if _, ok := want[id]; !ok {
			t.Error("extra missed : ",id)
		}
	}
	
	
	
}


