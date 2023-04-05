package gorules

import (
	"fmt"

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
		if len(node.Transitions) == 0 {
			results = append(results, node)
		}
		for _, prospect := range node.Transitions {
			isValid, err := valid(prospect, data)
			if err != nil {
				return nil, err
			}
			if isValid {
				results = append(results, prospect)
			}
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

func valid(node Node, data map[string]any) (valid bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			valid = false
			err = fmt.Errorf("error applying rules %+v with data %+v", node.Rules, data)
		}
	}()
	res, err := jsonlogic.ApplyInterface(node.Rules, data)
	if err != nil {
		return false, err
	}
	if asMap, ok := res.(map[string]any); ok && len(asMap) == 0 {
		return true, nil
	}
	if asBool, _ := res.(bool); asBool {
		return true, nil
	}
	return false, nil
}
