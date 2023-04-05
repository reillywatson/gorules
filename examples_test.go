package gorules_test

import (
	"fmt"

	"github.com/reillywatson/gorules"
)

func ExampleSolve_simple() {
	graph := []*gorules.Node{
		// a is always reachable, and a terminal node
		{Payload: "a"},
		// b is reachable if "foo" is true, and has a transition to c, which is always reachable.
		{Payload: "b", Rules: map[string]any{"var": []any{"foo"}}, Transitions: []*gorules.Node{{Payload: "c"}}},
	}
	got, err := gorules.Solve(graph, map[string]any{"foo": true})
	if err != nil {
		panic(err)
	}
	for _, node := range got {
		fmt.Println(node.Payload)
	}
	// Output: a
	// c
}
