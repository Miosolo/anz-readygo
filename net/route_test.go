package net

import (
	"math"
	"reflect"
	"testing"

	. "github.com/miosolo/readygo/io"
)

func TestRestContext_calcRoute(t *testing.T) {
	type args struct {
		initPoint  Asset
		sampleRate float64
	}
	tests := []struct {
		name              string
		r                 RestContext
		args              args
		wantFinalRoutePtr *Route
		wantErrCode       int
		wantErr           bool
	}{{
		name: "squre curve",
		args: args{
			initPoint: Asset{
				Name: "init point",
				Base: "base",
				Rx:   0,
				Ry:   0},
			sampleRate: 1.0},
		wantFinalRoutePtr: &Route{
			Sequence: []Checkpoint{
				Checkpoint{Name: "init point", Base: "base", Rx: 0, Ry: 0, IsPortal: false},
				Checkpoint{Name: "A", Base: "base", Rx: 1, Ry: 1, IsPortal: false, Weight: 1},
				Checkpoint{Name: "Meeting Room", Base: "base", Rx: 2, Ry: 2, IsPortal: true},
				Checkpoint{Name: "D", Base: "Meeting Room", Rx: 2, Ry: 3, IsPortal: false, Weight: 1},
				Checkpoint{Name: "Meeting Room", Base: "base", Rx: 2, Ry: 2, IsPortal: true},
				Checkpoint{Name: "B", Base: "base", Rx: 3, Ry: 1, IsPortal: false, Weight: 1},
				Checkpoint{Name: "C", Base: "base", Rx: 4, Ry: 0, IsPortal: false, Weight: 1}},
			Distance: 2 + 4*math.Sqrt(2)},
		wantErrCode: 200,
		wantErr:     false}}

	RCTest.InitEnv()
	RCTest.dbInsertSpace([]Space{
		Space{Name: "base", Base: "", Rx: 0, Ry: 0},
		Space{Name: "Meeting Room", Base: "base", Rx: 2, Ry: 2}})
	RCTest.dbInsertAsset([]Asset{
		Asset{Name: "A", Base: "base", Rx: 1, Ry: 1, Weight: 1},
		Asset{Name: "D", Base: "Meeting Room", Rx: 0, Ry: 1, Weight: 1},
		Asset{Name: "B", Base: "base", Rx: 3, Ry: 1, Weight: 1},
		Asset{Name: "C", Base: "base", Rx: 4, Ry: 0, Weight: 1}})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.InitEnv()
			if err := tt.r.InitEnv(); err != nil {
				panic("err initializing env")
			}
			gotFinalRoutePtr, gotErrCode, err := tt.r.calcRoute(tt.args.initPoint, tt.args.sampleRate)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestContext.calcRoute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotFinalRoutePtr, tt.wantFinalRoutePtr) {
				t.Errorf("RestContext.calcRoute() gotFinalRoutePtr = %v, want %v", gotFinalRoutePtr, tt.wantFinalRoutePtr)
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("RestContext.calcRoute() gotErrCode = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
	}

	RCTest.dbDeleteSpace("base")
}
