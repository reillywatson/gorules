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
	Transitions []Node
	// Weight defines the order in which results from Solve will be returned.
	Weight int
	// WeightRules is an optional set of rules to define the weight of this node.
	// A node with no WeightRules has a weight of `Weight`.
	WeightRules map[string]any
	Rules       map[string]any
}

func (n Node) String() string {
	return n.Id
}

type Result struct {
	Node   *Node
	Weight int
}

// Solve returns a list of terminal Nodes reachable from nodes, given the values in data.
// Solve returns an error if any of the nodes' rules are invalid JsonLogic definitions.
// Nodes are returned in descending order of weight.
func Solve(nodes []Node, data map[string]any) ([]Node, error) {
	for {
		var err error
		nodes, err = transitions(nodes, data)
		if err != nil {
			return nil, err
		}
		if allTerminal(nodes) {
			break
		}
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Weight > nodes[j].Weight
	})
	return nodes, nil
}

func transitions(startNodes []Node, data map[string]any) ([]Node, error) {
	var results []Node
	for _, node := range startNodes {
		isValid, err := valid(node, data)
		if err != nil {
			return nil, err
		}
		if !isValid {
			continue
		}
		node.Weight, err = weight(node.Weight, node.WeightRules, data)
		if err != nil {
			return nil, err
		}
		if len(node.Transitions) == 0 {
			results = append(results, node)
		}
		for _, prospect := range node.Transitions {
			isValid, err := valid(prospect, data)
			if err != nil {
				return nil, err
			}
			if !isValid {
				continue
			}
			prospect.Weight, err = weight(prospect.Weight, prospect.WeightRules, data)
			if err != nil {
				return nil, err
			}
			results = append(results, prospect)
		}
	}

	return results, nil
}

func allTerminal(nodes []Node) bool {
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

func valid(node Node, data map[string]any) (valid bool, err error) {
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
