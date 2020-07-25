package net

import (
	"testing"
)

func TestRestContext_InitEnv(t *testing.T) {
	tests := []struct {
		name    string
		r       RestContext
		wantErr bool
	}{{
		name:    "Test connection to server",
		r:       RestContext{},
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.InitEnv()
			if (err != nil) != tt.wantErr {
				t.Errorf("RestContext.InitEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
