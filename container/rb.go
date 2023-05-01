package main

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
)

type Direction uint8
type Color bool

const (
	Left  Direction = 0
	Right Direction = 1
)

const (
	Black Color = false
	Red   Color = true
)

type Comparable[T any] interface {
	Compare(T) int
}

type RBTree[K Comparable[K], V any] struct {
	Root *RBNode[K, V]

	// AfterMove gets called after a node got moved during a rotation, or after a node got deleted.
	AfterMove func(oldParent, node *RBNode[K, V])
}

type RBNode[K Comparable[K], V any] struct {
	parent   *RBNode[K, V]
	children [2]*RBNode[K, V]
	key      K
	value    V
	color    Color
}

func NewRBNode[K Comparable[K], V any](k K, v V) *RBNode[K, V] {
	return &RBNode[K, V]{
		key:   k,
		value: v,
	}
}

func (T *RBTree[K, V]) Search(k K) (node *RBNode[K, V], found bool, dir Direction) {
	if T.Root == nil {
		return nil, false, 0
	}

	x := T.Root
	for {
		switch k.Compare(x.key) {
		case -1:
			dir = Left
		case 0:
			return x, true, 0
		case 1:
			dir = Right
		}

		child := x.children[dir]
		if child == nil {
			return x, false, dir
		}
		x = child
	}
}

func (T *RBTree[K, V]) rotate(P *RBNode[K, V], dir Direction) *RBNode[K, V] {
	oldParent := P.parent

	G := P.parent
	S := P.children[1-dir]
	C := S.children[dir]
	P.children[1-dir] = C
	if C != nil {
		C.parent = P
	}
	S.children[dir] = P
	P.parent = S
	S.parent = G
	if G != nil {
		var child Direction
		if P == G.children[Right] {
			child = Right
		} else {
			child = Left
		}
		G.children[child] = S
	} else {
		T.Root = S
	}

	if T.AfterMove != nil {
		T.AfterMove(oldParent, P)
	}

	return S
}

func (T *RBTree[K, V]) Insert(k K, v V) *RBNode[K, V] {
	if T.Root == nil {
		N := NewRBNode(k, v)
		T.insert(N, nil, 0)
		return N
	}

	P, ok, dir := T.Search(k)
	if ok {
		P.value = v
		return P
	} else {
		N := NewRBNode(k, v)
		T.insert(N, P, dir)
		return N
	}
}

func (T *RBTree[K, V]) insert(N *RBNode[K, V], P *RBNode[K, V], dir Direction) {
	var G *RBNode[K, V]
	var U *RBNode[K, V]

	N.color = Red
	N.children[Left] = nil
	N.children[Right] = nil
	N.parent = P
	if P == nil {
		T.Root = N
		return
	}
	P.children[dir] = N

	for {
		if P.color == Black {
			return
		}

		G = P.parent
		if G == nil {
			P.color = Black
			return
		}

		dir = P.childDir()
		U = G.children[1-dir]
		if U == nil || U.color == Black {
			if N == P.children[1-dir] {
				T.rotate(P, dir)
				N = P
				P = G.children[dir]
			}

			T.rotate(G, 1-dir)
			P.color = Black
			G.color = Red
			return
		}

		P.color = Black
		U.color = Black
		G.color = Red
		N = G

		P = N.parent
		if P == nil {
			break
		}
	}
}

func (N *RBNode[K, V]) childDir() Direction {
	if N.parent.children[Right] == N {
		return Right
	} else {
		return Left
	}
}

func (N *RBNode[K, V]) Dot(w io.Writer, meta func(n *RBNode[K, V]) string) {
	p := func(s string) {
		w.Write([]byte(s))
		w.Write([]byte("\n"))
	}
	pf := func(f string, vs ...any) {
		fmt.Fprintf(w, f, vs...)
		w.Write([]byte("\n"))
	}

	var node func(N *RBNode[K, V])
	node = func(N *RBNode[K, V]) {
		var c string
		if N.color == Black {
			c = "black"
		} else {
			c = "red"
		}
		label := fmt.Sprintf("%v = %v", N.key, N.value)
		if meta != nil {
			label += "\n" + meta(N)
		}
		pf(`p%p [label="%s", color=%s];`, N, label, c)

		for i, child := range N.children {
			if child == nil {
				pf("p%pc%d [label=nil, style=invis];", N, i)
				pf("p%p -> p%pc%d [style=invis];", N, N, i)
			} else {
				node(child)
				pf("p%p -> p%p;", N, child)
			}
		}

	}

	p("digraph {")
	p("graph [ordering=out];")

	node(N)

	p("}")
}

type Int int

func (n Int) Compare(o Int) int {
	if n < o {
		return -1
	} else if n == o {
		return 0
	} else {
		return 1
	}
}

type Interval struct {
	Min, Max int
}

type Value struct {
	MaxSubtree int
	Value      string
}

func (ival Interval) Compare(oval Interval) int {
	if ival.Min < oval.Min {
		return -1
	} else if ival.Min > oval.Min {
		return 1
	} else {
		if ival.Max < oval.Max {
			return -1
		} else if ival.Max > oval.Max {
			return 1
		} else {
			return 0
		}
	}
}

type IntervalTree struct {
	RBTree[Interval, Value]
}

func (t *IntervalTree) Insert(min, max int, value string) *RBNode[Interval, Value] {
	n := t.RBTree.Insert(Interval{min, max}, Value{MaxSubtree: max, Value: value})
	t.updateAug(n.parent)
	return n
}

func (t *IntervalTree) updateAug(n *RBNode[Interval, Value]) bool {
	if n == nil {
		return false
	}

	old := n.value.MaxSubtree

	var vs [3]int
	for i := range vs[:2] {
		vs[i] = math.MinInt
	}
	vs[2] = n.key.Max

	for i, c := range n.children {
		if c != nil {
			vs[i] = c.value.MaxSubtree
		}
	}

	max := vs[0]
	for _, v := range vs[1:] {
		if v > max {
			max = v
		}
	}

	if max != old {
		n.value.MaxSubtree = max
		t.updateAug(n.parent)
		return true
	}

	return false
}

func main() {
	var t IntervalTree
	t.AfterMove = func(oldParent, node *RBNode[Interval, Value]) {
		println("hi")
		for t.updateAug(oldParent) || t.updateAug(node) {
			println("nice")
		}
		println("")
	}

	for i := 0; i < 100; i++ {
		var min, max int
		max = rand.Intn(500)
		for min > max {
			min = rand.Intn(500)
		}

		t.Insert(min, max, "")
	}

	t.Root.Dot(os.Stdout, func(n *RBNode[Interval, Value]) string {
		return fmt.Sprintf("aug = %d", n.value.MaxSubtree)
	})
}
