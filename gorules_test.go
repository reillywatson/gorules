package gorules

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func FuzzSolve(f *testing.F) {
	f.Add([]byte(`[{"Id": "a", "Transitions": [{"Id": "b", Rules: [{"var": ["b"]}]}]}]`), []byte(`{"b":true}`))
	f.Fuzz(func(t *testing.T, graphBytes, dataBytes []byte) {
		var graph []Node
		if err := json.Unmarshal(graphBytes, &graph); err != nil {
			return
		}
		var data map[string]any
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			return
		}
		Solve(graph, data)
	})
}

func TestSolve(t *testing.T) {
	tests := []struct {
		Start       []Node
		Data        map[string]any
		ExpectedIds []string
	}{
		{
			Start: []Node{
				{
					Id: "a",
					Transitions: []Node{
						{Id: "b"},
						{
							Id: "c", Transitions: []Node{
								{Id: "d"},
							},
						},
					},
				},
				{
					Id:    "e",
					Rules: mustParse(`{"var": ["is_smoker"]}`),
					Transitions: []Node{
						{Id: "f"},
						{
							Id: "g", Transitions: []Node{
								{Id: "h"},
							},
						},
					},
				},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: []string{"b", "d"},
		},
		{
			Start: []Node{
				{
					Id: "a",
					Transitions: []Node{
						{Id: "b"},
						{
							Id: "c", Transitions: []Node{
								{Id: "d"},
							},
						},
					},
				},
				{
					Id:    "e",
					Rules: mustParse(`{"var": ["is_smoker"]}`),
					Transitions: []Node{
						{Id: "f"},
						{
							Id: "g", Transitions: []Node{
								{Id: "h"},
							},
						},
					},
				},
			},
			Data:        map[string]any{"is_smoker": true},
			ExpectedIds: []string{"b", "d", "f", "h"},
		},
		{
			Start: []Node{
				{Id: "a"},
				{Id: "b", Rules: mustParse(`{"var": ["is_smoker"]}`)},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: []string{"a"},
		},
		{
			Start: []Node{
				{Id: "a", Rules: mustParse(`{"var": ["is_smoker"]}`)},
				{Id: "b", Rules: mustParse(`{"var": ["is_smoker"]}`)},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: nil,
		},
		{
			Start: []Node{
				{Id: "a", Rules: mustParse(`{"var": ["is_smoker"]}`)},
			},
			Data:        nil,
			ExpectedIds: nil,
		},
	}
	for _, test := range tests {
		got, err := Solve(test.Start, test.Data)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ids(got), test.ExpectedIds) {
			t.Errorf("expected %v, got %v", test.ExpectedIds, ids(got))
		}
	}
}

func TestBadInputsReturnErrors(t *testing.T) {
	tests := []struct {
		Start  []Node
		Data   map[string]any
		ExpErr error
	}{
		{
			Start: []Node{{Rules: map[string]any{"bad rule": 1}}},
		},
		{
			Start: []Node{{Transitions: []Node{{Rules: map[string]any{"bad rule": 1}}}}},
		},
	}
	for _, test := range tests {
		_, err := Solve(test.Start, test.Data)
		if err == nil {
			t.Error("got nil error, expected an error")
		}
	}
}

func TestLoopDoesntRunForever(t *testing.T) {
	a := Node{Id: "a"}
	b := Node{Id: "b"}
	c := Node{Id: "c"}
	// we can't really create a loop, because these aren't pointers!
	// When we assign b to a.Transitions it makes a copy, so when we
	// subsequently set b.Transitions it's operating on a different copy of that node.
	a.Transitions = []Node{b}
	b.Transitions = []Node{c}
	c.Transitions = []Node{a}
	done := make(chan bool)
	go func() {
		_, _ = Solve([]Node{a}, nil)
		done <- true
	}()
	select {
	case <-done:
		break
	case <-time.After(time.Second * 2):
		t.Fatal("graph with loop never returned")
	}
}

func ids(nodes []Node) []string {
	var res []string
	for _, n := range nodes {
		res = append(res, n.Id)
	}
	return res
}

func mustParse(rules string) map[string]any {
	var res map[string]any
	if err := json.Unmarshal([]byte(rules), &res); err != nil {
		panic(err)
	}
	return res
}