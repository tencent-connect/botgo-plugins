// Package base 本文件提供获取本机ip函数
package base

import "testing"

func TestGetLocalIP(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "c1", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetLocalIP()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLocalIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
