// package gorules is a rules engine. Given a set of input nodes and a data map, it will output a list of terminal nodes
// reachable from the input nodes, ordered by weight.
// Rules are implemented using JsonLogic (https://jsonlogic.com).

package gorules

import (
	"fmt"
	"sort"

	"github.com/diegoholiveira/jsonlogic"
)

// A Node represents a point in a graph.
type Node struct {
	// Payload is an arbitrary set of data associated with this Node.
	Payload any
	// Transitions is the list of Nodes that can be transitioned to from this node.
	// A node with no transitions is considered a terminal node.
	Transitions []*Node
	// Weight defines the order in which results from Solve will be returned.
	// If WeightRules are defined, results from Solve will populate Weight with the
	// calculated weight, otherwise Weight will be the weight as passed into Solve.
	Weight int
	// WeightRules is an optional set of rules to define the weight of this node.
	// A node with no WeightRules has a weight of `Weight`.
	// WeightRules are defined as JsonLogic maps (see https://jsonlogic.com).
	WeightRules map[string]any
	// Rules is an optional set of rules to define whether this node is reachable.
	// A node with no Rules is considered reachable.
	// Rules are defined as JsonLogic maps (see https://jsonlogic.com).
	Rules map[string]any
}

// ErrCycleDetected is returned if a graph is passed to Solve that has a cycle in it.
var ErrCycleDetected = fmt.Errorf("cycle detected in graph")

// Solve returns a list of terminal Nodes reachable from nodes, given the values in data.
// Solve returns an error if any of the nodes' rules are invalid JsonLogic definitions,
// or ErrCycleDetected if the graph contains a cycle.
// Nodes are returned in descending order of weight.
func Solve(nodes []*Node, data map[string]any) ([]*Node, error) {
	var err error
	nodes, err = bfs(nodes, data, valid)
	if err != nil {
		return nil, err
	}
	for _, n := range nodes {
		var err error
		if n.Weight, err = weight(n.Weight, n.WeightRules, data); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Weight > nodes[j].Weight
	})
	return nodes, nil
}

func bfs(start []*Node, data map[string]any, canVisit func(node *Node, data map[string]any) (bool, error)) ([]*Node, error) {
	var res []*Node
	visited := map[*Node]bool{}
	queue := start
	for _, s := range start {
		visited[s] = true
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		valid, err := canVisit(curr, data)
		if err != nil {
			return nil, err
		}
		if !valid {
			continue
		}
		if len(curr.Transitions) == 0 {
			res = append(res, curr)
		}
		for _, t := range curr.Transitions {
			if !visited[t] {
				visited[t] = true
				queue = append(queue, t)
			}
		}
	}

	return res, nil
}

func weight(weight int, rules map[string]any, data map[string]any) (val int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error applying rules %+v with data %+v", rules, data)
		}
	}()
	if len(rules) == 0 {
		return weight, nil
	}
	res, err := jsonlogic.ApplyInterface(rules, data)
	if err != nil {
		return 0, err
	}
	if asFloat, ok := res.(float64); ok {
		return int(asFloat), nil
	}
	return 0, fmt.Errorf("rule weight didn't return a number, got type %T", res)
}

func valid(node *Node, data map[string]any) (valid bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			valid = false
			err = fmt.Errorf("error applying rules %+v with data %+v", node.Rules, data)
		}
	}()
	if len(node.Rules) == 0 {
		return true, nil
	}
	res, err := jsonlogic.ApplyInterface(node.Rules, data)
	if err != nil {
		return false, err
	}
	if asBool, ok := res.(bool); ok {
		return asBool, nil
	}
	return false, fmt.Errorf("rule didn't return a boolean, got %T", res)
}
