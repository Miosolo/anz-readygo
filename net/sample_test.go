package net

import (
	"reflect"
	"testing"
)

func Test_sample(t *testing.T) {
	type args struct {
		wholeList []Asset
		rate      float64
	}
	tests := []struct {
		name                 string
		args                 args
		wantSampledIndexList []int
	}{{
		name: "Minimal",
		args: args{
			wholeList: []Asset{
				Asset{"A", "base", 0.4, 0.2, 1},
			},
			rate: 1.0,
		},
		wantSampledIndexList: []int{0},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotSampledIndexList := sample(tt.args.wholeList, tt.args.rate); !reflect.DeepEqual(gotSampledIndexList, tt.wantSampledIndexList) {
				t.Errorf("sample() = %v, want %v", gotSampledIndexList, tt.wantSampledIndexList)
			}
		})
	}
}
