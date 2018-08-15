package routing

import (
	//"time"
	
)

type tNode struct {
	Addr	string
	ID		Hash
	AddTime		int64
	
}

func NewNode(id Hash , addr string) *tNode {
	return &tNode{
		Addr:	addr,
		ID:		id,
	}
}

