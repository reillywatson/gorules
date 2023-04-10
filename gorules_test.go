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
	f.Add([]byte(`[{"Payload":"a","Transitions":[{"Payload":"b"},{"Payload":"c","Transitions":[{"Payload":"d"}]}]},{"Payload":"e","Transitions":[{"Payload":"f"},{"Payload":"g","Transitions":[{"Payload":"h"}]}],"Rules":{"var":["is_smoker"]}}]`), []byte(`{"is_smoker":false}`))
	f.Add([]byte(`[{"Payload":"a"},{"Payload":"b","Rules":{"var":["is_smoker"]}}]`), []byte(`{"is_smoker":false}`))
	f.Add([]byte(`[{"Payload":"a","Transitions":[{"Payload":"b","WeightRules":{"if":[{"var":["is_smoker"]},100,50]}},{"Payload":"c","Weight":75}]}]`), []byte(`{"is_smoker":true}`))
	f.Add([]byte(`[{"Payload":"a","Rules":{"and":[{"\u003c":[{"var":"temp"},110]},{"==":[{"var":"pie.filling"},"apple"]}]}}]`), []byte(`{"pie":{"filling":"apple"},"temp":100})`))
	f.Fuzz(func(t *testing.T, graphBytes, dataBytes []byte) {
		var graph []*Node[string]
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
		Start       []*Node[string]
		Data        map[string]any
		ExpectedIds []string
	}{
		{
			Start: []*Node[string]{
				{
					Payload: "a",
					Transitions: []*Node[string]{
						{Payload: "b"},
						{
							Payload: "c", Transitions: []*Node[string]{
								{Payload: "d"},
							},
						},
					},
				},
				{
					Payload: "e",
					Rules:   mustParse(`{"var": ["is_smoker"]}`),
					Transitions: []*Node[string]{
						{Payload: "f"},
						{
							Payload: "g", Transitions: []*Node[string]{
								{Payload: "h"},
							},
						},
					},
				},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: []string{"b", "d"},
		},
		{
			Start: []*Node[string]{
				{
					Payload: "a",
					Transitions: []*Node[string]{
						{Payload: "b", Weight: 10},
						{
							Payload: "c", Transitions: []*Node[string]{
								{Payload: "d"},
							},
						},
					},
				},
				{
					Payload: "e",
					Rules:   mustParse(`{"var": ["is_smoker"]}`),
					Transitions: []*Node[string]{
						{Payload: "f", Weight: 20},
						{
							Payload: "g", Transitions: []*Node[string]{
								{Payload: "h"},
							},
						},
					},
				},
			},
			Data:        map[string]any{"is_smoker": true},
			ExpectedIds: []string{"f", "b", "d", "h"},
		},
		{
			Start: []*Node[string]{
				{Payload: "a"},
				{Payload: "b", Rules: mustParse(`{"var": ["is_smoker"]}`)},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: []string{"a"},
		},
		{
			Start: []*Node[string]{
				{Payload: "a", Rules: mustParse(`{"var": ["is_smoker"]}`)},
				{Payload: "b", Rules: mustParse(`{"var": ["is_smoker"]}`)},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: nil,
		},
		{
			Start: []*Node[string]{
				{Payload: "a", Transitions: []*Node[string]{{Payload: "b", Rules: mustParse(`{"var": ["is_smoker"]}`)}}},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: nil,
		},
		{
			Start: []*Node[string]{
				{Payload: "a", Transitions: []*Node[string]{
					{Payload: "b", WeightRules: mustParse(`{"if" : [ {"var":["is_smoker"]}, 100, 50 ]}`)},
					{Payload: "c", Weight: 75},
				}},
			},
			Data:        map[string]any{"is_smoker": true},
			ExpectedIds: []string{"b", "c"},
		},
		{
			Start: []*Node[string]{
				{Payload: "a", Transitions: []*Node[string]{
					{Payload: "b", WeightRules: mustParse(`{"if" : [ {"var":["is_smoker"]}, 100, 50 ]}`)},
					{Payload: "c", Weight: 75},
				}},
			},
			Data:        map[string]any{"is_smoker": false},
			ExpectedIds: []string{"c", "b"},
		},
		{
			Start: []*Node[string]{
				{Payload: "a", Rules: mustParse(`{ "and" : [
					{"<" : [ { "var" : "temp" }, 110 ]},
					{"==" : [ { "var" : "pie.filling" }, "apple" ] }
				  ] }`)},
			},
			Data:        mustParse(`{ "temp" : 100, "pie" : { "filling" : "apple" } }`),
			ExpectedIds: []string{"a"},
		},
		{
			Start: []*Node[string]{
				{Payload: "a", Rules: mustParse(`{ "and" : [
					{"<" : [ { "var" : "temp" }, 110 ]},
					{"==" : [ { "var" : "pie.filling" }, "apple" ] }
				  ] }`)},
			},
			Data:        mustParse(`{ "temp" : 120, "pie" : { "filling" : "apple" } }`),
			ExpectedIds: nil,
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
	a := &Node[string]{Payload: "a"}
	b := &Node[string]{Payload: "b"}
	c := &Node[string]{Payload: "c"}
	d := &Node[string]{Payload: "d"}
	a.Transitions = []*Node[string]{b, c}
	b.Transitions = []*Node[string]{d}
	c.Transitions = []*Node[string]{d}
	got, err := Solve([]*Node[string]{a}, nil)
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
		Start  []*Node[string]
		Data   map[string]any
		ExpErr error
	}{
		{
			Start: []*Node[string]{{Rules: map[string]any{"bad rule": 1}}},
		},
		{
			// returns a non-boolean
			Start: []*Node[string]{{Rules: mustParse(`{"if" : [ {"var":["is_smoker"]}, 100, 50 ]}`)}},
		},
		{
			Start: []*Node[string]{{WeightRules: mustParse(`{"var": ["is_smoker"]}`)}},
		},
		{
			Start: []*Node[string]{{WeightRules: map[string]any{"bad rule": 1}}},
		},
		{
			Start: []*Node[string]{{Transitions: []*Node[string]{{Rules: map[string]any{"bad rule": 1}}}}},
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
	a := &Node[string]{Payload: "a"}
	b := &Node[string]{Payload: "b"}
	c := &Node[string]{Payload: "c"}
	a.Transitions = []*Node[string]{b}
	b.Transitions = []*Node[string]{c}
	c.Transitions = []*Node[string]{a}
	doneChan := make(chan bool)
	go func() {
		_, _ = Solve([]*Node[string]{a}, nil)
		doneChan <- true
	}()
	select {
	case <-doneChan:
	case <-time.After(time.Second * 2):
		t.Fatal("graph with loop never returned")
	}
}

func TestDynamicVariable(t *testing.T) {
	graph := []*Node[string]{
		{Payload: "a", Rules: mustParse(`
{
  "==": [
    { "var": [
      {"var": ["foo"] }
     ]},
   10
  ]
}`)},
	}
	got, err := Solve(graph, mustParse(`{"foo": "bar", "bar": 10}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range got {
		fmt.Println(node.Payload)
	}
}

func BenchmarkSolve(b *testing.B) {
	graph := []*Node[string]{
		{
			Payload: "a",
			Transitions: []*Node[string]{
				{Payload: "b"},
				{
					Payload: "c", Transitions: []*Node[string]{
						{Payload: "d"},
					},
				},
			},
		},
		{
			Payload: "e",
			Rules:   mustParse(`{"var": ["is_smoker"]}`),
			Transitions: []*Node[string]{
				{Payload: "f"},
				{
					Payload: "g", Transitions: []*Node[string]{
						{Payload: "h"},
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

func ids[T any](nodes []*Node[T]) []T {
	var res []T
	for _, n := range nodes {
		res = append(res, n.Payload)
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
