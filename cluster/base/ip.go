// Package base 本文件提供获取本机ip函数
package base

import (
	"fmt"
	"net"
	"strings"
)

// GetLocalIP 获取本机IP
func GetLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			// 跳过loopback实例
			continue
		}
		if strings.Index(iface.Name, "eth") == 0 {
			// 返回首个eth网口的ip
			return getIP(iface)
		}
	}
	return "", fmt.Errorf("no valid iface:%v", interfaces)
}

// getIP 获取网口ip
func getIP(iface net.Interface) (string, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	for _, v := range addrs {
		ipNet, ok := v.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.To4() != nil ||
			ipNet.IP.To16() != nil {
			if ip := ipNet.IP.String(); ip != "" {
				return ip, nil
			}
		}
	}
	return "", fmt.Errorf("iface have no valid ip:%v", iface)
}
