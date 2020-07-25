package net

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/gomodule/redigo/redis"
	"go.mongodb.org/mongo-driver/bson"
)

func TestRestContext_dbGetAsset(t *testing.T) {
	type args struct {
		name      string
		base      string
		cacheFlag bool
	}
	tests := []struct {
		name        string
		r           RestContext
		args        args
		wantResult  *Asset
		wantErrCode int
		wantErr     bool
	}{{
		name:        "get A",
		args:        args{name: "A", base: "test", cacheFlag: false},
		wantResult:  &Asset{Name: "A", Base: "test", Rx: 1, Ry: 1, Weight: 1},
		wantErrCode: 200,
		wantErr:     false}, {
		name:        "get wrong D",
		args:        args{name: "D", base: "test", cacheFlag: false},
		wantResult:  nil,
		wantErrCode: 404,
		wantErr:     true}, {
		name:        "get right D",
		args:        args{name: "D", base: "test/Meeting Room", cacheFlag: false},
		wantErrCode: 200,
		wantResult:  &Asset{Name: "D", Base: "test/Meeting Room", Rx: 2, Ry: 3, Weight: 1},
		wantErr:     false}}

	RCTest.InitEnv()
	RCTest.dbInsertSpace([]Space{
		Space{Name: "test", Base: "", Rx: 0, Ry: 0},
		Space{Name: "test/Meeting Room", Base: "test", Rx: 0, Ry: 0}})
	RCTest.dbInsertAsset([]Asset{
		Asset{Name: "A", Base: "test", Rx: 1, Ry: 1, Weight: 1},
		Asset{Name: "D", Base: "test/Meeting Room", Rx: 2, Ry: 3, Weight: 1}})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r = RCTest
			gotResult, gotErrCode, err := tt.r.dbGetAsset(tt.args.name, tt.args.base, tt.args.cacheFlag)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestContext.dbGetAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("RestContext.dbGetAsset() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("RestContext.dbGetAsset() gotErrCode = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
	}
	RCTest.dbDeleteSpace("test")
}

func TestRestContext_dbInsertSpace(t *testing.T) {
	type args struct {
		list []Space
	}
	tests := []struct {
		name        string
		r           RestContext
		args        args
		wantErrCode int
		wantErr     bool
	}{{
		name:        "insert a test space",
		args:        args{[]Space{Space{Name: "test-1", Base: "test", Rx: 1, Ry: 1}}},
		wantErrCode: http.StatusCreated,
		wantErr:     false}, {
		name: "insert multiple space",
		args: args{[]Space{
			Space{Name: "test-2", Base: "test", Rx: 2, Ry: 2},
			Space{Name: "test-3", Base: "test", Rx: 3, Ry: 3},
			Space{Name: "test-4", Base: "test", Rx: 4, Ry: 4}}},
		wantErrCode: http.StatusCreated,
		wantErr:     false}, {
		name:        "insert a replicated space",
		args:        args{[]Space{Space{Name: "test-1", Base: "test", Rx: 2, Ry: 2}}},
		wantErrCode: http.StatusConflict,
		wantErr:     true}}

	RCTest.InitEnv()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r = RCTest
			gotErrCode, err := tt.r.dbInsertSpace(tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestContext.dbInsertSpace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("RestContext.dbInsertSpace() = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
	}
	RCTest.dbDeleteSpace("test")
}

func TestRestContext_dbInsertAsset(t *testing.T) {
	type args struct {
		list []Asset
	}
	tests := []struct {
		name        string
		r           RestContext
		args        args
		wantErrCode int
		wantErr     bool
	}{{
		name:        "test single insert",
		args:        args{list: []Asset{Asset{Name: "test-A", Base: "test", Rx: 1, Ry: 1, Weight: 1}}},
		wantErrCode: http.StatusCreated,
		wantErr:     false}, {
		name: "test multiple insert",
		args: args{list: []Asset{
			Asset{Name: "test-B", Base: "test", Rx: 1, Ry: 1, Weight: 1},
			Asset{Name: "test-C", Base: "test", Rx: 1, Ry: 1, Weight: 1},
			Asset{Name: "test-D", Base: "test", Rx: 1, Ry: 1, Weight: 1},
			Asset{Name: "test-E", Base: "test", Rx: 1, Ry: 1, Weight: 1}}},
		wantErrCode: http.StatusCreated,
		wantErr:     false}, {
		name:        "test duplicate insert",
		args:        args{list: []Asset{Asset{Name: "test-A", Base: "test", Rx: 1, Ry: 1, Weight: 1}}},
		wantErrCode: http.StatusConflict,
		wantErr:     true}}

	RCTest.InitEnv()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r = RCTest
			gotErrCode, err := tt.r.dbInsertAsset(tt.args.list)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestContext.dbInsertAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("RestContext.dbInsertAsset() = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
	}
	RCTest.dbDeleteSpace("test")
}

func TestRestContext_dbGetSpace(t *testing.T) {
	type args struct {
		name      string
		cacheFlag bool
	}
	tests := []struct {
		name        string
		r           RestContext
		args        args
		wantResult  *Space
		wantErrCode int
		wantErr     bool
	}{{
		name:        "valid space",
		args:        args{name: "test", cacheFlag: false},
		wantResult:  &Space{Name: "test", Base: "", Rx: 0, Ry: 0},
		wantErrCode: 200,
		wantErr:     false}, {
		name:        "not existing space w/ cache",
		args:        args{name: "nonsense", cacheFlag: false},
		wantResult:  nil,
		wantErrCode: 404,
		wantErr:     true}}

	RCTest.InitEnv()
	RCTest.dbInsertSpace([]Space{Space{Name: "test", Base: "", Rx: 0, Ry: 0}})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r = RCTest
			gotResult, gotErrCode, err := tt.r.dbGetSpace(tt.args.name, tt.args.cacheFlag)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestContext.dbGetSpace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("RestContext.dbGetSpace() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("RestContext.dbGetSpace() gotErrCode = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
		if tt.name == "not existing space w/ cache" {
			c := tt.r.redisConnPool.Get()
			sp := Space{}
			data, _ := redis.Bytes(c.Do("GET", "space-"+tt.args.name))
			if err := json.Unmarshal(data, &sp); err == nil {
				t.Error("dirty cache")
			}
		}
	}
	RCTest.dbDeleteSpace("test")
}

func TestRestContext_dbDeleteAsset(t *testing.T) {
	type args struct {
		name string
		base string
	}
	tests := []struct {
		name        string
		r           RestContext
		args        args
		wantErrCode int
		wantErr     bool
	}{{name: "delete one",
		args:        args{name: "test-A", base: "test"},
		wantErrCode: 200,
		wantErr:     false}, {
		name:        "delete another",
		args:        args{name: "test-C", base: "test"},
		wantErrCode: 200,
		wantErr:     false}, {
		name:        "delete nonsense",
		args:        args{name: "test-F", base: "test"},
		wantErrCode: 404,
		wantErr:     true}}

	RCTest.InitEnv()
	RCTest.dbInsertAsset([]Asset{
		Asset{Name: "test-A", Base: "test", Rx: 1, Ry: 1, Weight: 1},
		Asset{Name: "test-C", Base: "test", Rx: 1, Ry: 1, Weight: 1}})

	for _, tt := range tests {
		tt.r = RCTest
		t.Run(tt.name, func(t *testing.T) {
			gotErrCode, err := tt.r.dbDeleteAsset(tt.args.name, tt.args.base)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestContext.dbDeleteAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("RestContext.dbDeleteAsset() = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
	}
}

func TestRestContext_dbDeleteSpace(t *testing.T) {
	type args struct {
		rootSpace string
	}
	tests := []struct {
		name        string
		r           RestContext
		args        args
		wantErrCode int
		wantErr     bool
	}{{
		name:        "delete subtree",
		args:        args{rootSpace: "test/1"},
		wantErrCode: 200,
		wantErr:     false}, {
		name:        "delete root node",
		args:        args{rootSpace: "test"},
		wantErrCode: 200,
		wantErr:     false}, {
		name:        "delete nonsense space",
		args:        args{rootSpace: "test/0"},
		wantErrCode: 404,
		wantErr:     true}}

	RCTest.InitEnv()
	RCTest.dbInsertSpace([]Space{
		Space{Name: "test", Base: "", Rx: 0, Ry: 0},
		Space{Name: "test/1", Base: "test", Rx: 0, Ry: 0},
		Space{Name: "test/2", Base: "test", Rx: 0, Ry: 0},
		Space{Name: "test/3", Base: "test/2", Rx: 0, Ry: 0},
		Space{Name: "test/4", Base: "test/3", Rx: 0, Ry: 0}})
	RCTest.dbInsertAsset([]Asset{
		Asset{Name: "test/A", Base: "test", Rx: 1, Ry: 1, Weight: 1},
		Asset{Name: "test/B", Base: "test/1", Rx: 1, Ry: 1, Weight: 1},
		Asset{Name: "test/C", Base: "test/1", Rx: 1, Ry: 1, Weight: 1},
		Asset{Name: "test/D", Base: "test/2", Rx: 1, Ry: 1, Weight: 1},
		Asset{Name: "test/E", Base: "test/4", Rx: 1, Ry: 1, Weight: 1}})

	for _, tt := range tests {
		tt.r = RCTest
		t.Run(tt.name, func(t *testing.T) {
			gotErrCode, err := tt.r.dbDeleteSpace(tt.args.rootSpace)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestContext.dbDeleteSpace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("RestContext.dbDeleteSpace() = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
	}
}

func TestRestContext_dbUpdateAsset(t *testing.T) {
	type args struct {
		name  string
		base  string
		toSet bson.D
	}
	tests := []struct {
		name            string
		r               RestContext
		args            args
		wantNewAssetPtr *Asset
		wantErrCode     int
		wantErr         bool
	}{{
		name:            "bisic test",
		args:            args{name: "test-update-1", base: "test", toSet: bson.D{{"$set", bson.D{{"rx", 2}, {"ry", 2}}}}},
		wantNewAssetPtr: &Asset{Name: "test-update-1", Base: "test", Rx: 2, Ry: 2, Weight: 1},
		wantErrCode:     200,
		wantErr:         false}, {
		name:            "not found test",
		args:            args{name: "test-update-2", base: "test", toSet: bson.D{{"$set", bson.D{{"rx", 2}, {"ry", 2}}}}},
		wantNewAssetPtr: nil,
		wantErrCode:     404,
		wantErr:         true}}

	RCTest.InitEnv()
	RCTest.dbInsertSpace([]Space{Space{Name: "test", Base: "", Rx: 0, Ry: 0}})
	RCTest.dbInsertAsset([]Asset{Asset{Name: "test-update-1", Base: "test", Rx: 0, Ry: 0, Weight: 1}})
	for _, tt := range tests {
		tt.r = RCTest
		t.Run(tt.name, func(t *testing.T) {
			gotNewAssetPtr, gotErrCode, err := tt.r.dbUpdateAsset(tt.args.name, tt.args.base, tt.args.toSet)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestContext.dbUpdateAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotNewAssetPtr, tt.wantNewAssetPtr) {
				t.Errorf("RestContext.dbUpdateAsset() gotNewAssetPtr = %v, want %v", gotNewAssetPtr, tt.wantNewAssetPtr)
			}
			if gotErrCode != tt.wantErrCode {
				t.Errorf("RestContext.dbUpdateAsset() gotErrCode = %v, want %v", gotErrCode, tt.wantErrCode)
			}
		})
	}
	RCTest.dbDeleteSpace("test")
}

func TestRestContext_UnloadDemoData(t *testing.T) {
	tests := []struct {
		name string
		r    RestContext
	}{{name: "unload sub of webtest"}}

	RCTest.InitEnv()
	for _, tt := range tests {
		tt.r = RCTest
		t.Run(tt.name, func(t *testing.T) {
			tt.r.UnloadDemoData()
		})
	}
}
