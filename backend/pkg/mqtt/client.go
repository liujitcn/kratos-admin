package mqtt

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/liujitcn/kratos-kit/broker"
	"github.com/liujitcn/kratos-kit/transport/mqtt"
)

// Registration 描述一个由宿主业务提供的 MQTT 主题订阅。
type Registration = mqtt.Registration

// NewRegistration 创建一个 MQTT 主题订阅注册项。
func NewRegistration(topic string, handler broker.Handler, binder broker.Binder, options ...broker.SubscribeOption) Registration {
	return mqtt.NewRegistration(topic, handler, binder, options...)
}

// Client 提供其他业务项目复用的 MQTT 发布和订阅能力。
type Client struct {
	server *mqtt.Server
}

type clientHolder struct {
	client *Client
}

var defaultClient atomic.Pointer[clientHolder]

// NewClient 创建绑定到宿主 MQTT 服务的业务客户端。
func NewClient(server *mqtt.Server) *Client {
	return &Client{server: server}
}

// Register 将业务订阅注册到宿主 MQTT 服务。
func (c *Client) Register(registration Registration) error {
	if c == nil || c.server == nil {
		return fmt.Errorf("mqtt client: server is nil")
	}
	return c.server.Register(registration)
}

// Subscribe 使用主题、消息处理器和绑定器注册一个业务订阅。
func (c *Client) Subscribe(topic string, handler broker.Handler, binder broker.Binder, options ...broker.SubscribeOption) error {
	return c.Register(NewRegistration(topic, handler, binder, options...))
}

// RegisterSubscriptions 批量注册业务订阅到当前 MQTT 服务。
func (c *Client) RegisterSubscriptions(registrations ...Registration) error {
	for _, registration := range registrations {
		if err := c.Register(registration); err != nil {
			return err
		}
	}
	return nil
}

// Publish 发布一个业务消息，具体编码由宿主 MQTT 服务配置决定。
func (c *Client) Publish(ctx context.Context, topic string, body any, options ...broker.PublishOption) error {
	if c == nil || c.server == nil {
		return fmt.Errorf("mqtt client: server is nil")
	}
	instance := c.server.Broker()
	if instance == nil {
		return fmt.Errorf("mqtt client: broker is nil")
	}
	return instance.Publish(ctx, topic, broker.NewMessage(body), options...)
}

// SetDefaultClient 设置当前进程共享的 MQTT 客户端。
func SetDefaultClient(client *Client) {
	if client == nil {
		defaultClient.Store(nil)
		return
	}
	defaultClient.Store(&clientHolder{client: client})
}

// DefaultClient 返回当前进程共享的 MQTT 客户端。
func DefaultClient() (*Client, error) {
	holder := defaultClient.Load()
	if holder == nil || holder.client == nil {
		return nil, fmt.Errorf("mqtt client: default client is not initialized")
	}
	return holder.client, nil
}

// Register 使用当前进程共享的 MQTT 客户端注册一个业务订阅。
func Register(registration Registration) error {
	client, err := DefaultClient()
	if err != nil {
		return err
	}
	return client.Register(registration)
}

// Subscribe 使用当前进程共享的 MQTT 客户端注册一个业务订阅。
func Subscribe(topic string, handler broker.Handler, binder broker.Binder, options ...broker.SubscribeOption) error {
	client, err := DefaultClient()
	if err != nil {
		return err
	}
	return client.Subscribe(topic, handler, binder, options...)
}

// Publish 使用当前进程共享的 MQTT 客户端发布一个业务消息。
func Publish(ctx context.Context, topic string, body any, options ...broker.PublishOption) error {
	client, err := DefaultClient()
	if err != nil {
		return err
	}
	return client.Publish(ctx, topic, body, options...)
}

// RegisterSubscriptions 批量注册业务订阅到当前进程共享的 MQTT 客户端。
func RegisterSubscriptions(registrations ...Registration) error {
	client, err := DefaultClient()
	if err != nil {
		return err
	}
	return client.RegisterSubscriptions(registrations...)
}
