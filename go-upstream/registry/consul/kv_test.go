package consul

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/yunfeiyang1916/toolkit/logging"
)

func Test_watchKV(t *testing.T) {
	type args struct {
		client *api.Client
		path   string
		config chan string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			watchKV(logging.New(), tt.args.client, tt.args.path, tt.args.config)
		})
	}
}

func Test_getKV(t *testing.T) {
	type args struct {
		client    *api.Client
		key       string
		waitIndex uint64
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   uint64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := getKV(tt.args.client, tt.args.key, tt.args.waitIndex)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getKV() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getKV() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_putKV(t *testing.T) {
	type args struct {
		client *api.Client
		key    string
		value  string
		index  uint64
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := putKV(tt.args.client, tt.args.key, tt.args.value, tt.args.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("putKV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("putKV() = %v, want %v", got, tt.want)
			}
		})
	}
}
