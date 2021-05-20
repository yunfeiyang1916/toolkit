package sql

import (
	"reflect"
	"testing"
)

func TestNewGroup(t *testing.T) {
	type args struct {
		name   string
		master string
		slaves []string
	}
	tests := []struct {
		name    string
		args    args
		want    *Group
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGroup(SQLGroupConfig{
				Name:   tt.args.name,
				Master: tt.args.master,
				Slaves: tt.args.slaves,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroup_Master(t *testing.T) {
	type fields struct {
		name    string
		master  *Client
		replica []*Client
		next    uint64
		total   uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   *Client
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Group{
				name:    tt.fields.name,
				master:  tt.fields.master,
				replica: tt.fields.replica,
				next:    tt.fields.next,
				total:   tt.fields.total,
			}
			if got := g.Master(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Group.Master() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroup_Slave(t *testing.T) {
	type fields struct {
		name    string
		master  *Client
		replica []*Client
		next    uint64
		total   uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   *Client
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Group{
				name:    tt.fields.name,
				master:  tt.fields.master,
				replica: tt.fields.replica,
				next:    tt.fields.next,
				total:   tt.fields.total,
			}
			if got := g.Slave(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Group.Slave() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroup_Instance(t *testing.T) {
	type fields struct {
		name    string
		master  *Client
		replica []*Client
		next    uint64
		total   uint64
	}
	type args struct {
		isMaster bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Client
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Group{
				name:    tt.fields.name,
				master:  tt.fields.master,
				replica: tt.fields.replica,
				next:    tt.fields.next,
				total:   tt.fields.total,
			}
			if got := g.Instance(tt.args.isMaster); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Group.Instance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseConnAddress(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   int
		want2   int
		want3   int
		wantErr bool
	}{
		{
			"test1",
			args{"admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8"},
			"admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com:3306)/live_serviceinfo?charset=utf8",
			15,
			0,
			1800,
			false,
		},
		{
			"test2",
			args{"admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com)/live_serviceinfo?charset=utf8&max_idle=10&max_active=200&max_lifetime_sec=3600"},
			"admin_user:ar46yJv34jfd@tcp(rm-2zej5vr9490158hv0.mysql.rds.aliyuncs.com:3306)/live_serviceinfo?charset=utf8",
			10,
			200,
			3600,
			false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3, err := parseConnAddress(tt.args.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConnAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseConnAddress() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseConnAddress() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("parseConnAddress() got2 = %v, want %v", got2, tt.want2)
			}
			if got3 != tt.want3 {
				t.Errorf("parseConnAddress() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}

func Test_openDB(t *testing.T) {
	type args struct {
		name    string
		address string
	}
	tests := []struct {
		name    string
		args    args
		want    *Client
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := openDB(tt.args.name, tt.args.address, 1, "", "")
			if (err != nil) != tt.wantErr {
				t.Errorf("openDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("openDB() = %v, want %v", got, tt.want)
			}
		})
	}
}
