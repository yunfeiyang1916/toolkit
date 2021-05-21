package consul

import (
	"testing"

	"github.com/yunfeiyang1916/toolkit/go-upstream/registry"
)

func Test_checkChanged(t *testing.T) {
	type args struct {
		new  []*registry.Cluster
		last []*registry.Cluster
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test-equal",
			args{
				[]*registry.Cluster{&registry.Cluster{Name: "test", Endpoints: []registry.Endpoint{
					registry.Endpoint{
						ID:   "id2",
						Addr: "addr2",
						Port: 1234,
						Tags: []string{"a=b", "c=d"},
					},
					registry.Endpoint{
						ID:   "id",
						Addr: "addr",
						Port: 1234,
						Tags: []string{"a=b", "c=d"},
					},
				}}},
				[]*registry.Cluster{&registry.Cluster{Name: "test", Endpoints: []registry.Endpoint{
					registry.Endpoint{
						ID:   "id",
						Addr: "addr",
						Port: 1234,
						Tags: []string{"a=b", "c=d"},
					},
					registry.Endpoint{
						ID:   "id2",
						Addr: "addr2",
						Port: 1234,
						Tags: []string{"a=b", "c=d"},
					},
				}}},
			},
			true,
		},
		{"test-equal",
			args{
				[]*registry.Cluster{&registry.Cluster{Name: "test", Endpoints: []registry.Endpoint{
					registry.Endpoint{
						ID:   "id",
						Addr: "addr",
						Port: 1234,
						Tags: []string{"a=b", "c=d"},
					},
					registry.Endpoint{
						ID:   "id2",
						Addr: "addr2",
						Port: 1234,
						Tags: []string{"c=d", "a=b"},
					},
				}}},
				[]*registry.Cluster{&registry.Cluster{Name: "test", Endpoints: []registry.Endpoint{
					registry.Endpoint{
						ID:   "id",
						Addr: "addr",
						Port: 1234,
						Tags: []string{"a=b", "c=d"},
					},
					registry.Endpoint{
						ID:   "id2",
						Addr: "addr2",
						Port: 1234,
						Tags: []string{"a=b", "c=d"},
					},
				}}},
			},
			false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// if got := checkChanged(tt.args.new, tt.args.last); got != tt.want {
			// 	t.Errorf("checkChanged() = %v, want %v", got, tt.want)
			// }
		})
	}
}
