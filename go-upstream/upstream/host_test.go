package upstream

import (
	"reflect"
	"testing"
)

func TestNewHost(t *testing.T) {
	type args struct {
		address string
		weight  uint32
		meta    map[string]string
	}
	tests := []struct {
		name string
		args args
		want *Host
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHost(tt.args.address, tt.args.weight, tt.args.meta); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHost_Address(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
			}
			if got := h.Address(); got != tt.want {
				t.Errorf("Host.Address() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHost_Meta(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// // meta: tt.fields.meta,
			}
			if got := h.Meta(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Host.Meta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHost_HealthFlagClear(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	type args struct {
		flag HealthFlag
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      bool
		healtWant bool
	}{
		// TODO: Add test cases.
		{"clear1", fields{"127.0.0.1:8080", [4]uint32{0, 0, 0, 0}, 0, 1, uint32(Unknown), nil}, args{flag: FailedActiveHC}, true, true},
		{"clear2", fields{"127.0.0.1:8080", [4]uint32{1, 1, 0, 0}, 0, 1, uint32(Unknown), nil}, args{flag: FailedActiveHC}, true, false},
		{"clear3", fields{"127.0.0.1:8080", [4]uint32{0, 1, 0, 0}, 0, 1, uint32(Unknown), nil}, args{flag: FailedDetectorCheck}, true, true},
		{"clear4", fields{"127.0.0.1:8080", [4]uint32{0, 1, 0, 0}, 0, 1, uint32(Unknown), nil}, args{flag: FailedActiveHC}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// // meta: tt.fields.meta,
			}
			h.HealthFlagClear(tt.args.flag)
			if got := h.HealthFlagGet(tt.args.flag); !reflect.DeepEqual(got, tt.want) || !reflect.DeepEqual(h.Healthy(), tt.healtWant) {
				t.Errorf("%q Host.HealthFlagGet() = %v, want %v, %v health want %v", tt.name, got, tt.want, h.Healthy(), tt.healtWant)
			}
		})
	}
}

func TestHost_HealthFlagGet(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	type args struct {
		flag HealthFlag
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantRes bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// meta: tt.fields.meta,
			}
			if gotRes := h.HealthFlagGet(tt.args.flag); gotRes != tt.wantRes {
				t.Errorf("Host.HealthFlagGet() = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func TestHost_HealthFlagSet(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	type args struct {
		flag HealthFlag
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// meta: tt.fields.meta,
			}
			h.HealthFlagSet(tt.args.flag)
		})
	}
}

func TestHost_Healthy(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	tests := []struct {
		name       string
		fields     fields
		wantHealth bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// meta: tt.fields.meta,
			}
			if gotHealth := h.Healthy(); gotHealth != tt.wantHealth {
				t.Errorf("Host.Healthy() = %v, want %v", gotHealth, tt.wantHealth)
			}
		})
	}
}

func TestHost_GetActiveHealthFailureType(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		wantTp ActiveHalthFailureType
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// meta: tt.fields.meta,
			}
			if gotTp := h.GetActiveHealthFailureType(); gotTp != tt.wantTp {
				t.Errorf("Host.GetActiveHealthFailureType() = %v, want %v", gotTp, tt.wantTp)
			}
		})
	}
}

func TestHost_SetActiveHealthFailureType(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	type args struct {
		tp ActiveHalthFailureType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// meta: tt.fields.meta,
			}
			h.SetActiveHealthFailureType(tt.args.tp)
		})
	}
}

func TestHost_Weight(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	tests := []struct {
		name       string
		fields     fields
		wantWeight uint32
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// meta: tt.fields.meta,
			}
			if gotWeight := h.Weight(); gotWeight != tt.wantWeight {
				t.Errorf("Host.Weight() = %v, want %v", gotWeight, tt.wantWeight)
			}
		})
	}
}

func TestHost_SetWeight(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	type args struct {
		new uint32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// meta: tt.fields.meta,
			}
			h.SetWeight(tt.args.new)
		})
	}
}

func TestHost_Used(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	tests := []struct {
		name     string
		fields   fields
		wantUsed bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// meta: tt.fields.meta,
			}
			if gotUsed := h.Used(); gotUsed != tt.wantUsed {
				t.Errorf("Host.Used() = %v, want %v", gotUsed, tt.wantUsed)
			}
		})
	}
}

func TestHost_SetUsed(t *testing.T) {
	type fields struct {
		address                 string
		healthFlag              [4]uint32
		weight                  uint32
		used                    uint32
		activeHealthFailureType uint32
		meta                    map[string]string
	}
	type args struct {
		new bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{
				address:                 tt.fields.address,
				healthFlag:              tt.fields.healthFlag,
				weight:                  tt.fields.weight,
				used:                    tt.fields.used,
				activeHealthFailureType: tt.fields.activeHealthFailureType,
				// meta: tt.fields.meta,
			}
			h.SetUsed(tt.args.new)
		})
	}
}
