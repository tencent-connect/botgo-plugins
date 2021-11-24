package etcd

import (
	"reflect"
	"testing"
)

var (
	testInsID          = "test"
	testInsClusterName = "testInsCluster"
)

func Test_newInstanceWithID(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		args    args
		want    *Instance
		wantErr bool
	}{
		{
			name:    "c1",
			args:    args{id: ""},
			want:    nil,
			wantErr: true,
		}, {
			name:    "c2",
			args:    args{id: testInsID},
			want:    &Instance{id: testInsID},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newInstanceWithID(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("newInstanceWithID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newInstanceWithID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newInstance(t *testing.T) {
	type args struct {
		clusterName string
		name        string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "c1",
			args:    args{clusterName: ""},
			wantErr: true,
		}, {
			name:    "c2",
			args:    args{clusterName: testInsClusterName},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newInstance(tt.args.clusterName, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("newInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
