package route

import (
	"testing"

	. "github.com/miosolo/readygo/io"
)

func TestDrawRoute(t *testing.T) {
	type args struct {
		cpList []Checkpoint
	}
	tests := []struct {
		name        string
		args        args
		wantErrCode int
		wantErr     bool
	}{{
		name: "square curve & subspace",
		args: args{cpList: []Checkpoint{
			Checkpoint{Name: "init point", Base: "base", Rx: 0, Ry: 0, IsPortal: false},
			Checkpoint{Name: "A", Base: "base", Rx: 1, Ry: 1, IsPortal: false, Weight: 1},
			Checkpoint{Name: "Meeting Room", Base: "base", Rx: 2, Ry: 2, IsPortal: true},
			Checkpoint{Name: "D", Base: "Meeting Room", Rx: 2, Ry: 3, IsPortal: false, Weight: 1},
			Checkpoint{Name: "Meeting Room", Base: "base", Rx: 2, Ry: 2, IsPortal: true},
			Checkpoint{Name: "B", Base: "base", Rx: 3, Ry: 1, IsPortal: false, Weight: 1},
			Checkpoint{Name: "C", Base: "base", Rx: 4, Ry: 0, IsPortal: false, Weight: 1}}},
		wantErrCode: 200,
		wantErr:     false}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErrCode, err := DrawRoute(tt.args.cpList)
			if (err != nil) != tt.wantErr {
				t.Errorf("DrawRoute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("DrawRoute() gotErrCode = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
	}
}
