package io

import (
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestReadCsvByName(t *testing.T) {
	type args struct {
		fileLoc string
	}
	tests := []struct {
		name          string
		args          args
		wantCpListPtr *[]Checkpoint
		wantErrCode   int
		wantErr       bool
	}{{
		name: "Simple",
		args: args{strings.Join([]string{
			os.Getenv("GOPATH"), "src", "github.com", "miosolo", "readygo", "test", "test_location.csv",
		}, string(os.PathSeparator))}, // in test/test.location.csv
		wantCpListPtr: &[]Checkpoint{
			Checkpoint{"A", "base", 0.4, 0.2, false, 2},
			Checkpoint{"B", "base", 3.5, 2, false, 1},
			Checkpoint{"C", "base", 2.6, 5.9, false, 1},
			Checkpoint{"D", "base", 3, 2.1, true, 0},
			Checkpoint{"E", "D", 2, 1.5, false, 1},
			Checkpoint{"F", "D", 1.7, 1, false, 1},
		},
		wantErrCode: http.StatusCreated,
		wantErr:     false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := os.Stat(tt.args.fileLoc); err != nil { // test file not ready
				t.Skip("The test csv file not ready, skip")
			}
			gotCpListPtr, gotErrCode, err := ReadCsvByName(tt.args.fileLoc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadCsvByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotCpListPtr, tt.wantCpListPtr) {
				t.Errorf("ReadCsvByName() gotCpListPtr = %v, want %v", gotCpListPtr, tt.wantCpListPtr)
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("ReadCsvByName() gotErrCode = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
	}
}
