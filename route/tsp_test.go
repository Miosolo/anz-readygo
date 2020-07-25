package route

import (
	"reflect"
	"sync"
	"testing"

	. "github.com/miosolo/readygo/io"
)

func TestTSP(t *testing.T) {
	type args struct {
		cpList      []Checkpoint
		Portal      Checkpoint
		circuitFlag bool
		result      *Route
		wgPtr       *sync.WaitGroup
	}
	tests := []struct {
		name       string
		args       args
		wantRoutes []Route
	}{{
		name: "Stright line",
		args: args{
			cpList: []Checkpoint{
				Checkpoint{"A", "base", 0, 1, false, 1},
				Checkpoint{"B", "base", 0, 2, false, 1},
				Checkpoint{"C", "base", 0, 3, false, 1},
				Checkpoint{"D", "base", 0, 4, false, 1},
			},
			Portal:      Checkpoint{"init", "base", 0, 0, false, 1},
			circuitFlag: false,
			result:      &Route{}},
		wantRoutes: []Route{
			Route{
				Sequence: []Checkpoint{
					Checkpoint{"init", "base", 0, 0, false, 1},
					Checkpoint{"A", "base", 0, 1, false, 1},
					Checkpoint{"B", "base", 0, 2, false, 1},
					Checkpoint{"C", "base", 0, 3, false, 1},
					Checkpoint{"D", "base", 0, 4, false, 1}},
				Distance: 4}},
	}, {
		name: "Square",
		args: args{
			cpList: []Checkpoint{
				Checkpoint{"D", "base", 0, 0, false, 1},
				Checkpoint{"C", "base", 1, 0, false, 1},
				Checkpoint{"B", "base", 1, 1, false, 1},
				Checkpoint{"A", "base", 0, 1, false, 1},
			},
			Portal:      Checkpoint{"init", "base", 0, 0, false, 1},
			circuitFlag: true,
			result:      &Route{}},
		wantRoutes: []Route{
			Route{
				Sequence: []Checkpoint{
					Checkpoint{"init", "base", 0, 0, false, 1},
					Checkpoint{"D", "base", 0, 0, false, 1},
					Checkpoint{"C", "base", 1, 0, false, 1},
					Checkpoint{"B", "base", 1, 1, false, 1},
					Checkpoint{"A", "base", 0, 1, false, 1},
					Checkpoint{"init", "base", 0, 0, false, 1}},
				Distance: 4},
			Route{
				Sequence: []Checkpoint{
					Checkpoint{"init", "base", 0, 0, false, 1},
					Checkpoint{"A", "base", 0, 1, false, 1},
					Checkpoint{"B", "base", 1, 1, false, 1},
					Checkpoint{"C", "base", 1, 0, false, 1},
					Checkpoint{"D", "base", 0, 0, false, 1},
					Checkpoint{"init", "base", 0, 0, false, 1}},
				Distance: 4}}}}

	for _, tt := range tests {
		revertedWantList := make([]Checkpoint, len(tt.args.cpList))
		for i := len(tt.args.cpList) - 1; i >= 0; i-- {
			revertedWantList = append(revertedWantList, tt.args.cpList[i])
		}
		t.Run(tt.name, func(t *testing.T) {
			TSP(tt.args.cpList, tt.args.Portal, tt.args.circuitFlag, tt.args.result)
			match := false
			for _, r := range tt.wantRoutes {
				match = match || reflect.DeepEqual(r, *tt.args.result)
			}
			if !match {
				t.Errorf("tsp() gotRoute = %v, want %v", *tt.args.result, tt.wantRoutes)
			}
		})
	}
}
