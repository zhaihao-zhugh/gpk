package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

type Logger interface {
	Debug(args ...any)
	Info(args ...any)
	Error(args ...any)
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

type MQConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Durable  bool   `json:"durable"`
}

type MQ struct {
	config          *MQConfig
	Conn            *amqp.Connection
	channels        []*MQChannel
	isConnected     bool
	reconnectDelay  time.Duration
	notifyConnClose chan *amqp.Error
	// notifyChanClose chan *amqp.Error
	mutex sync.Mutex
}

type ChannelConfig struct {
	Type     string   `json:"type"`
	Exchange string   `json:"exchange"`
	Queue    string   `json:"queue"`
	Key      []string `json:"key"`
	Durable  bool     `json:"durable"`
}

type MQChannel struct {
	Type            int //  1: consumer   2: producer
	MQ              *MQ
	Channel         *amqp.Channel
	config          *ChannelConfig
	notifyChanClose chan *amqp.Error
	RdData          <-chan amqp.Delivery
	handler         func(body []byte)
}

func (cfg *MQConfig) address() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.Username, cfg.Password, cfg.Host, cfg.Port)
}

func New(cfg *MQConfig) *MQ {
	return &MQ{
		config:         cfg,
		reconnectDelay: 20 * time.Second,
	}
}

func (mq *MQ) Connect() error {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	if mq.isConnected {
		return nil
	}

	conn, err := amqp.Dial(mq.config.address())
	if err != nil {
		return err
	}
	defer conn.Close()
	mq.Conn = conn

	// 监听连接关闭
	mq.notifyConnClose = make(chan *amqp.Error)
	mq.Conn.NotifyClose(mq.notifyConnClose)
	mq.isConnected = true
	log.Println("MQ已连接")

	// 启动监听重连
	go mq.handleReconnect()
	return nil
}

// 处理重连
func (mq *MQ) handleReconnect() {
	for {
		select {
		case <-mq.notifyConnClose:
			mq.isConnected = false
			log.Println("MQ 连接已关闭，尝试重连...")
			mq.reconnect()
		}
	}
}

// 重连
func (mq *MQ) reconnect() {
	for {
		mq.mutex.Lock()
		if mq.isConnected {
			mq.mutex.Unlock()
			return
		}
		mq.mutex.Unlock()
		log.Printf("尝试重新连接到 MQ...")
		err := mq.Connect()
		if err != nil {
			log.Printf("重连失败: %s，%s 后重试", err, mq.reconnectDelay)
			time.Sleep(mq.reconnectDelay)
			continue
		}
		return
	}
}

// 关闭连接
func (mq *MQ) Close() error {
	if !mq.isConnected {
		return nil
	}

	mq.mutex.Lock()
	defer mq.mutex.Unlock()

	if mq.Conn != nil {
		return mq.Conn.Close()
	}

	mq.isConnected = false
	return nil
}

// 检查连接状态
func (mq *MQ) IsConnected() bool {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()
	return mq.isConnected
}

func (mq *MQ) NewConsumer(config *ChannelConfig, handler func(body []byte)) (*MQChannel, error) {
	c := &MQChannel{
		Type: 1,
	}
	if ch, err := mq.Conn.Channel(); err != nil {
		return c, err
	} else {
		c.MQ = mq
		c.Channel = ch
		c.config = config
		// 定义 exchange
		err = ch.ExchangeDeclare(
			config.Exchange,   // name
			config.Type,       // type
			mq.config.Durable, // durable
			false,             // auto-deleted
			false,             // internal
			false,             // no-wait
			nil,               // arguments
		)
		if err != nil {
			return c, err
		}

		// 定义 queue
		q, err := ch.QueueDeclare(
			config.Queue,      // name
			mq.config.Durable, // durable
			true,              // delete when unused
			false,             // exclusive 是否私有
			false,             // no-wait
			nil,               // arguments
		)
		if err != nil {
			return c, err
		}

		if len(config.Key) == 0 {
			err = ch.QueueBind(
				q.Name,          // queue name
				"",              // routing key
				config.Exchange, // exchange
				false,           //	noWait
				nil,
			)
			if err != nil {
				return c, err
			}
		}

		for _, v := range config.Key {
			err = ch.QueueBind(
				q.Name,          // queue name
				v,               // routing key
				config.Exchange, // exchange
				false,           //	noWait
				nil,
			)
			if err != nil {
				return c, err
			}
		}

		//订阅消息，并不是把mq的消息直接写到msgs，不需要死循环订阅，订阅之后mq有消息就会往msgs里写
		msgs, err := ch.Consume(
			q.Name, // queue
			"",     // consumer
			true,   // auto ack
			false,  // exclusive
			false,  // no local
			false,  // no wait
			nil,    // args
		)

		if err != nil {
			return c, err
		}
		c.RdData = msgs
		c.notifyChanClose = make(chan *amqp.Error)
		c.Channel.NotifyClose(c.notifyChanClose)
		c.handler = handler
		go c.HandleMsg()
		go c.handleReconnect()
	}
	log.Printf("消费者创建成功(%s)\n", config.Queue)
	return c, nil
}

func (mq *MQ) NewProducer(config *ChannelConfig) (*MQChannel, error) {
	p := &MQChannel{
		Type: 2,
	}
	if ch, err := mq.Conn.Channel(); err != nil {
		return p, err
	} else {
		p.MQ = mq
		p.Channel = ch
		p.config = config
		err = ch.ExchangeDeclare(
			config.Exchange,   // name
			config.Type,       // type
			mq.config.Durable, // durable
			false,             // auto-deleted
			false,             // internal
			false,             // no-wait
			nil,               // arguments
		)
		if err != nil {
			return p, err
		}
	}
	log.Printf("发布者创建成功(%s)\n", config.Exchange)
	return p, nil
}

func (mq *MQ) AddChannel(c *MQChannel) {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()
	mq.channels = append(mq.channels, c)
}

func (c *MQChannel) HandleMsg() {
	for msg := range c.RdData {
		c.handler(msg.Body)
	}
}

func (c *MQChannel) PublishMsg(data interface{}) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = c.Channel.Publish(
		c.config.Exchange,
		"",
		false, //mandatory：true：如果exchange根据自身类型和消息routeKey无法找到一个符合条件的queue，那么会调用basic.return方法将消息返还给生产者。false：出现上述情形broker会直接将消息扔掉
		false, //如果exchange在将消息route到queue(s)时发现对应的queue上没有消费者，那么这条消息不会放入队列中。当与消息routeKey关联的所有queue(一个或多个)都没有消费者时，该消息会通过basic.return方法返还给生产者。
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        buf,
		})
	if err != nil {
		return err
	}
	return nil
}

func (c *MQChannel) PublishMsgWithKey(key string, data interface{}) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = c.Channel.Publish(
		c.config.Exchange,
		key,
		false, //mandatory：true：如果exchange根据自身类型和消息routeKey无法找到一个符合条件的queue，那么会调用basic.return方法将消息返还给生产者。false：出现上述情形broker会直接将消息扔掉
		false, //如果exchange在将消息route到queue(s)时发现对应的queue上没有消费者，那么这条消息不会放入队列中。当与消息routeKey关联的所有queue(一个或多个)都没有消费者时，该消息会通过basic.return方法返还给生产者。
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        buf,
		})
	return err
}

// 处理重连
func (c *MQChannel) handleReconnect() {
	for {
		select {
		case <-c.notifyChanClose:
			log.Printf("通道 %s 已关闭，尝试重连...\n", c.config.Exchange)
			c.reconnect()
		}
	}
}

// 重连
func (c *MQChannel) reconnect() {
	for {
		if c.MQ.IsConnected() {
			c.MQ.
		} else {
			log.Printf("MQ连接已关闭\n")
			return
		}
	}
}
