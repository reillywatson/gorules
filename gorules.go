package gorules

import (
	"fmt"
	"sort"

	"github.com/diegoholiveira/jsonlogic"
)

// A Node represents a point in a graph.
type Node struct {
	Id string
	// Payload is an arbitrary set of data associated with this Node.
	Payload any
	// Transitions is the list of Nodes that can be transitioned to from this node.
	// A node with no transitions is considered a terminal node.
	Transitions []*Node
	// Weight defines the order in which results from Solve will be returned.
	Weight int
	// WeightRules is an optional set of rules to define the weight of this node.
	// A node with no WeightRules has a weight of `Weight`.
	WeightRules map[string]any
	Rules       map[string]any

	// for cycle detection
	parents []*Node
}

type Result struct {
	Node   *Node
	Weight int
}

var ErrCycleDetected = fmt.Errorf("cycle detected in graph")

// Solve returns a list of terminal Nodes reachable from nodes, given the values in data.
// Solve returns an error if any of the nodes' rules are invalid JsonLogic definitions,
// or ErrCycleDetected if the graph contains a cycle.
// Nodes are returned in descending order of weight.
func Solve(nodes []*Node, data map[string]any) ([]*Node, error) {
	for i := 0; i < 10; i++ {
		var err error
		nodes, err = transitions(nodes, data)
		if err != nil {
			return nil, err
		}
		if allTerminal(nodes) {
			break
		}
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

func transitions(startNodes []*Node, data map[string]any) ([]*Node, error) {
	var results []*Node
	for i := 0; i < len(startNodes); i++ {
		node := startNodes[i]
		isValid, err := valid(node.Rules, data)
		if err != nil {
			return nil, err
		}
		if !isValid {
			continue
		}
		if len(node.Transitions) == 0 {
			results = append(results, node)
		}
		for j := 0; j < len(node.Transitions); j++ {
			prospect := node.Transitions[j]
			if contains(node.parents, prospect) {
				return nil, ErrCycleDetected
			}
			isValid, err := valid(prospect.Rules, data)
			if err != nil {
				return nil, err
			}
			if !isValid {
				continue
			}
			prospect.parents = append(append(prospect.parents, node), node.parents...)
			results = append(results, prospect)
		}
	}
	results = removeDuplicates(results)
	return results, nil
}

func allTerminal(nodes []*Node) bool {
	for _, n := range nodes {
		if len(n.Transitions) > 0 {
			return false
		}
	}
	return true
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

func valid(rules map[string]any, data map[string]any) (valid bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			valid = false
			err = fmt.Errorf("error applying rules %+v with data %+v", rules, data)
		}
	}()
	if len(rules) == 0 {
		return true, nil
	}
	res, err := jsonlogic.ApplyInterface(rules, data)
	if err != nil {
		return false, err
	}
	if asBool, ok := res.(bool); ok {
		return asBool, nil
	}
	return false, fmt.Errorf("rule didn't return a boolean, got %T", res)
}

func contains[T comparable](list []T, val T) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}

func removeDuplicates[T comparable](list []T) []T {
	if list == nil {
		return nil
	}
	res := make([]T, 0, len(list))
	seen := map[T]bool{}
	for _, v := range list {
		if seen[v] {
			continue
		}
		seen[v] = true
		res = append(res, v)
	}
	return res
}
