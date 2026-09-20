// Package registry 负责把 dextea-xos 注册到 Nacos，并在退出时注销。
package registry

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/wilson-lyc/dextea-v3-xos/internal/config"
)

type Registrar struct {
	client naming_client.INamingClient
	cfg    config.NacosConfig
	ip     string
	port   uint64
}

func Register(cfg config.NacosConfig, listenAddr string) (*Registrar, error) {
	ip, port, err := endpoint(listenAddr, cfg.InstanceIP)
	if err != nil {
		return nil, err
	}
	clientCfg := constant.NewClientConfig(
		constant.WithNamespaceId(cfg.NamespaceID),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithUsername(cfg.Username),
		constant.WithPassword(cfg.Password),
	)
	serverCfg := constant.NewServerConfig(cfg.ServerAddr, cfg.ServerPort)
	client, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  clientCfg,
		ServerConfigs: []constant.ServerConfig{*serverCfg},
	})
	if err != nil {
		return nil, fmt.Errorf("创建 nacos 客户端失败: %w", err)
	}
	ok, err := client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        port,
		ServiceName: cfg.ServiceName,
		GroupName:   cfg.GroupName,
		ClusterName: cfg.ClusterName,
		Weight:      cfg.Weight,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"protocol": "grpc"},
	})
	if err != nil || !ok {
		client.CloseClient()
		return nil, fmt.Errorf("注册到 nacos 失败: err=%v ok=%v", err, ok)
	}
	log.Printf("[info] registered %s (%s:%d) to nacos %s:%d/%s",
		cfg.ServiceName, ip, port, cfg.ServerAddr, cfg.ServerPort, cfg.GroupName)
	return &Registrar{client: client, cfg: cfg, ip: ip, port: port}, nil
}

func (r *Registrar) Deregister() {
	if r == nil || r.client == nil {
		return
	}
	defer r.client.CloseClient()
	ok, err := r.client.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          r.ip,
		Port:        r.port,
		ServiceName: r.cfg.ServiceName,
		GroupName:   r.cfg.GroupName,
		Cluster:     r.cfg.ClusterName,
		Ephemeral:   true,
	})
	if err != nil || !ok {
		log.Printf("[warn] deregister dextea-xos from nacos: err=%v ok=%v", err, ok)
		return
	}
	log.Printf("[info] deregistered dextea-xos from nacos")
}

func endpoint(addr, configuredIP string) (string, uint64, error) {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("解析监听地址 %s 失败: %w", addr, err)
	}
	port, err := strconv.ParseUint(portText, 10, 64)
	if err != nil || port == 0 {
		return "", 0, fmt.Errorf("解析监听端口 %s 失败", portText)
	}
	if ip := strings.TrimSpace(configuredIP); ip != "" {
		host = ip
	} else if host == "" || host == "0.0.0.0" || host == "::" {
		conn, err := net.Dial("udp", "8.8.8.8:80")
		if err == nil {
			defer conn.Close()
			if udpAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
				host = udpAddr.IP.String()
			}
		}
		if host == "" {
			addrs, listErr := net.InterfaceAddrs()
			if listErr != nil {
				return "", 0, fmt.Errorf("获取本机 IP 失败: %w", listErr)
			}
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
					host = ipNet.IP.String()
					break
				}
			}
		}
	}
	if host == "" {
		return "", 0, fmt.Errorf("未找到可用于注册 Nacos 的本机 IP")
	}
	return host, port, nil
}
