package grpcx

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"time"
	"webook-grpc/pkg/loggerx"
)

// Server grpc服务器，包含了与etcd交互的逻辑
type Server struct {
	*grpc.Server                   // grpc服务
	Port         int               // 服务监听的端口
	EtcdTTL      int64             // 租期
	EtcdClient   *clientv3.Client  // etcd客户端
	etcdManager  endpoints.Manager // etcd管理器，用于管理etcd服务
	etcdKey      string            // 服务在etcd中的唯一标识
	cancel       func()            // 用于取消续约
	Name         string            // 服务名称
	L            loggerx.Logger
}

// Serve 启动服务器并且阻塞
func (s *Server) Serve() error {
	// 创建一个context，用于控制服务续租
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	// 将服务监听的端口，转为string
	port := strconv.Itoa(s.Port)
	// 创建了一个监听器，用于监听指定的端口
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	// 将创建的服务，注册到etcd中
	err = s.register(ctx, port)
	if err != nil {
		return err
	}
	// 启动grpc服务，并监听传入的连接
	return s.Server.Serve(l)
}

// register 将服务注册到etcd中，并设置续租
func (s *Server) register(ctx context.Context, port string) error {
	cli := s.EtcdClient
	// 创建一个etcd管理器，用于管理etcd服务
	manager, err := endpoints.NewManager(cli, "service/"+s.Name)
	if err != nil {
		return err
	}
	s.etcdManager = manager
	// key，服务的唯一标识
	s.etcdKey = "service/" + s.Name + "/" + "localhost"
	// 服务的地址
	addr := "localhost" + ":" + port
	// 设置租期
	leaseResp, err := cli.Grant(ctx, s.EtcdTTL)
	// 开启续租
	//	  参数2：租期的ID
	//    返回值1：是一个管道，用来接收续租的结果
	ch, err := cli.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		return err
	}
	go func() {
		// 当调用cancel时，通道就会被关闭，然后就会退出这个循环
		for chResp := range ch {
			s.L.Debug("续约：", loggerx.String("resp", chResp.String()))
		}
	}()
	// 将服务注册到etcd中
	//	 如果key存在，则更新，否则创建
	err = manager.AddEndpoint(ctx, s.etcdKey, endpoints.Endpoint{
		Addr: addr,
	}, clientv3.WithLease(leaseResp.ID))
	return err
}

// Close 关闭服务
func (s *Server) Close() error {
	// 取消续租
	s.cancel()
	if s.etcdManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// 将服务从etcd中删除
		err := s.etcdManager.DeleteEndpoint(ctx, s.etcdKey)
		if err != nil {
			return err
		}
	}
	// 关闭etcd客户端
	err := s.EtcdClient.Close()
	if err != nil {
		return err
	}
	// 优雅退出grpc服务器
	s.Server.GracefulStop()
	return nil
}
