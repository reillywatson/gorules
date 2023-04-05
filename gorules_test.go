package gorules

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func FuzzSolve(f *testing.F) {
	f.Add([]byte(`[{"Id": "a", "Transitions": [{"Id": "b", Rules: [{"var": ["b"]}]}]}]`), []byte(`{"b":true}`))
	f.Fuzz(func(t *testing.T, graphBytes, dataBytes []byte) {
		var graph []*Node
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
		Start       []*Node
		Data        map[string]any
		ExpectedIds []string
	}{
		{
			Start: []*Node{
				{
					Id: "a",
					Transitions: []*Node{
						{Id: "b"},
						{
							Id: "c", Transitions: []*Node{
								{Id: "d"},
							},
						},
					},
				},
				{
					Id:    "e",
					Rules: mustParse(`{"var": ["is_smoker"]}`),
					Transitions: []*Node{
						{Id: "f"},
						{
							Id: "g", Transitions: []*Node{
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
			Start: []*Node{
				{
					Id: "a",
					Transitions: []*Node{
						{Id: "b", Weight: 10},
						{
							Id: "c", Transitions: []*Node{
								{Id: "d"},
							},
						},
					},
				},
				{
					Id:    "e",
					Rules: mustParse(`{"var": ["is_smoker"]}`),
					Transitions: []*Node{
						{Id: "f", Weight: 20},
						{
							Id: "g", Transitions: []*Node{
								{Id: "h"},
							},
						},
					},
				},
			},
			Data:        map[string]any{"is_smoker": true},
			ExpectedIds: []string{"f", "b", "d", "h"},
		},
		{
			Start: []*Node{
				{Id: "a"},
				{Id: "b", Rules: mustParse(`{"var": ["is_smoker"]}`)},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: []string{"a"},
		},
		{
			Start: []*Node{
				{Id: "a", Rules: mustParse(`{"var": ["is_smoker"]}`)},
				{Id: "b", Rules: mustParse(`{"var": ["is_smoker"]}`)},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: nil,
		},
		{
			Start: []*Node{
				{Id: "a", Transitions: []*Node{
					{Id: "b", WeightRules: mustParse(`{"if" : [ {"var":["is_smoker"]}, 100, 50 ]}`)},
					{Id: "c", Weight: 75},
				}},
			},
			Data:        map[string]any{"is_smoker": true},
			ExpectedIds: []string{"b", "c"},
		},
		{
			Start: []*Node{
				{Id: "a", Transitions: []*Node{
					{Id: "b", WeightRules: mustParse(`{"if" : [ {"var":["is_smoker"]}, 100, 50 ]}`)},
					{Id: "c", Weight: 75},
				}},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: []string{"c", "b"},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i+1), func(t *testing.T) {
			got, err := Solve(test.Start, test.Data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ids(got), test.ExpectedIds) {
				t.Errorf("expected %v, got %v", test.ExpectedIds, ids(got))
			}
		})
	}
}

func TestGraphWithMergingBranches(t *testing.T) {
	a := &Node{Id: "a"}
	b := &Node{Id: "b"}
	c := &Node{Id: "c"}
	d := &Node{Id: "d"}
	a.Transitions = []*Node{b, c}
	b.Transitions = []*Node{d}
	c.Transitions = []*Node{d}
	got, err := Solve([]*Node{a}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("expected one result, got %d", len(got))
	}
	if got[0] != d {
		t.Errorf("expected d, got %v", got[0])
	}
}

func TestBadInputsReturnErrors(t *testing.T) {
	tests := []struct {
		Start  []*Node
		Data   map[string]any
		ExpErr error
	}{
		{
			Start: []*Node{{Rules: map[string]any{"bad rule": 1}}},
		},
		{
			Start: []*Node{{WeightRules: mustParse(`{"var": ["is_smoker"]}`)}},
		},
		{
			Start: []*Node{{WeightRules: map[string]any{"bad rule": 1}}},
		},
		{
			Start: []*Node{{Transitions: []*Node{{Rules: map[string]any{"bad rule": 1}}}}},
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
	a := &Node{Id: "a"}
	b := &Node{Id: "b"}
	c := &Node{Id: "c"}
	a.Transitions = []*Node{b}
	b.Transitions = []*Node{c}
	c.Transitions = []*Node{a}
	errChan := make(chan error)
	go func() {
		_, err := Solve([]*Node{a}, nil)
		errChan <- err
	}()
	select {
	case err := <-errChan:
		if err == nil {
			t.Errorf("Expected error, got none")
		}
	case <-time.After(time.Second * 2):
		t.Fatal("graph with loop never returned")
	}
}

func BenchmarkSolve(b *testing.B) {
	graph := []*Node{
		{
			Id: "a",
			Transitions: []*Node{
				{Id: "b"},
				{
					Id: "c", Transitions: []*Node{
						{Id: "d"},
					},
				},
			},
		},
		{
			Id:    "e",
			Rules: mustParse(`{"var": ["is_smoker"]}`),
			Transitions: []*Node{
				{Id: "f"},
				{
					Id: "g", Transitions: []*Node{
						{Id: "h"},
					},
				},
			},
		},
	}
	data := map[string]any{"is_smoker": true, "foo": "bar", "some_other_val": true, "hey": "ya"}
	for i := 0; i < b.N; i++ {
		_, err := Solve(graph, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func ids(nodes []*Node) []string {
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
