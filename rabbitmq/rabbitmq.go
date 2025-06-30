package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

type Logger interface {
	Debug(args ...any)
	Info(args ...any)
	Error(args ...any)
}

type MQConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Durable  bool   `json:"durable"`
}

type ChannelConfig struct {
	Type     string   `json:"type"`
	Exchange string   `json:"exchange"`
	Queue    string   `json:"queue"`
	Key      []string `json:"key"`
	Durable  bool     `json:"durable"`
}

type MQ struct {
	config      *MQConfig
	Conn        *amqp.Connection
	consumers   []*Consumer
	producers   []*Producer
	NotifyClose chan *amqp.Error
}

type Consumer struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Config  *ChannelConfig
	// RdData  <-chan amqp.Delivery
	handler func(body []byte)
}

type Producer struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Config  *ChannelConfig
}

const (
	reconnectDelay = 15 * time.Second
)

func DefaultChannelConfig(exchange, queue string) *ChannelConfig {
	return &ChannelConfig{
		Type:     "topic",
		Exchange: exchange,
		Queue:    queue,
	}
}

func NewChannelConfig(t, exchange, queue string, key []string, durable bool) *ChannelConfig {
	return &ChannelConfig{
		Type:     t,
		Exchange: exchange,
		Queue:    queue,
		Key:      key,
		Durable:  durable,
	}
}

func listenClose(mq *MQ) {
	e := <-mq.NotifyClose
	log.Println(e)
	mq.Conn.Close()
	mq.reconnect()
	log.Println("重连成功...")
}

func (cfg *MQConfig) address() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.Username, cfg.Password, cfg.Host, cfg.Port)
}

func New(cfg *MQConfig) *MQ {
	mq := &MQ{
		config:      cfg,
		NotifyClose: make(chan *amqp.Error),
	}

	c, err := amqp.Dial(mq.config.address())
	if err != nil {
		log.Panic(err)
	}
	log.Println("MQ连接成功")
	c.NotifyClose(mq.NotifyClose)
	mq.Conn = c
	go listenClose(mq)
	return mq
}

func (mq *MQ) reconnect() {
	t := time.NewTimer(reconnectDelay)
	for {
		<-t.C
		if conn, err := amqp.Dial(mq.config.address()); err != nil {
			log.Println(err)
			t.Reset(reconnectDelay)
		} else {
			mq.NotifyClose = make(chan *amqp.Error)
			conn.NotifyClose(mq.NotifyClose)
			mq.Conn = conn

			for _, c := range mq.consumers {
				c.Channel.Close()
				c.Conn = mq.Conn
				c.Reset()
			}
			for _, p := range mq.producers {
				p.Channel.Close()
				p.Conn = mq.Conn
				p.Reset()
			}
			go listenClose(mq)
			return
		}
	}
}

func (mq *MQ) NewConsumer(config *ChannelConfig, handler func(body []byte)) *Consumer {
	c := &Consumer{}
	if ch, err := mq.Conn.Channel(); err != nil {
		log.Panic(err)
	} else {
		c.Conn = mq.Conn
		c.Channel = ch
		c.Config = config
		c.Config.Durable = mq.config.Durable

		// 定义 exchange
		err = ch.ExchangeDeclare(
			c.Config.Exchange, // name
			c.Config.Type,     // type
			c.Config.Durable,  // durable
			false,             // auto-deleted
			false,             // internal
			false,             // no-wait
			nil,               // arguments
		)
		if err != nil {
			log.Panic(err)
		}

		// 定义 queue
		q, err := ch.QueueDeclare(
			c.Config.Queue,   // name
			c.Config.Durable, // durable
			true,             // delete when unused
			false,            // exclusive 是否私有
			false,            // no-wait
			nil,              // arguments
		)
		if err != nil {
			log.Panic(err)
		}

		if len(c.Config.Key) == 0 {
			err = ch.QueueBind(
				q.Name,            // queue name
				"",                // routing key
				c.Config.Exchange, // exchange
				false,             //	noWait
				nil,
			)
			if err != nil {
				log.Panic(err)
			}
		}

		for _, v := range c.Config.Key {
			err = ch.QueueBind(
				q.Name,            // queue name
				v,                 // routing key
				c.Config.Exchange, // exchange
				false,             //	noWait
				nil,
			)
			if err != nil {
				log.Panic(err)
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
			log.Panic(err)
		}
		// c.RdData = msgs
		c.handler = handler
		mq.consumers = append(mq.consumers, c)
		go c.HandleMsg(msgs)
	}
	log.Println(c.Config.Queue + " 创建成功")
	return c
}

func (mq *MQ) NewProducer(config *ChannelConfig) *Producer {
	p := &Producer{}
	if ch, err := mq.Conn.Channel(); err != nil {
		log.Panic(err)
	} else {
		p.Conn = mq.Conn
		p.Config = config
		p.Config.Durable = mq.config.Durable
		p.Channel = ch
		err = ch.ExchangeDeclare(
			p.Config.Exchange, // name
			p.Config.Type,     // type
			p.Config.Durable,  // durable
			false,             // auto-deleted
			false,             // internal
			false,             // no-wait
			nil,               // arguments
		)
		if err != nil {
			log.Panic(err)
		}
		mq.producers = append(mq.producers, p)
	}
	log.Println(p.Config.Queue + " 创建成功")
	return p
}

func (c *Consumer) Reset() {
	t := time.NewTimer(reconnectDelay)
	for {
		<-t.C
		if ch, err := c.Conn.Channel(); err != nil {
			log.Println("创建通道失败: ", err)
			t.Reset(reconnectDelay)
		} else {
			c.Channel = ch
			// 定义 exchange
			err = ch.ExchangeDeclare(
				c.Config.Exchange, // name
				c.Config.Type,     // type
				c.Config.Durable,  // durable
				false,             // auto-deleted
				false,             // internal
				false,             // no-wait
				nil,               // arguments
			)
			if err != nil {
				log.Println("ExchangeDeclare error: ", err)
				t.Reset(reconnectDelay)
				continue
			}
			// 定义 queue
			q, err := ch.QueueDeclare(
				c.Config.Queue,   // name
				c.Config.Durable, // durable
				true,             // delete when unused
				false,            // exclusive 是否私有
				false,            // no-wait
				nil,              // arguments
			)
			if err != nil {
				log.Println("QueueDeclare error: ", err)
				t.Reset(reconnectDelay)
				continue
			}

			for _, v := range c.Config.Key {
				err = ch.QueueBind(
					q.Name,            // queue name
					v,                 // routing key
					c.Config.Exchange, // exchange
					false,             //	noWait
					nil,
				)
				if err != nil {
					log.Println("QueueBind error: ", err)
					t.Reset(reconnectDelay)
					continue
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
				log.Println("Consume error: ", err)
				t.Reset(reconnectDelay)
				continue
			}
			// c.RdData = msgs
			go c.HandleMsg(msgs)
			return
		}
	}
}

func (c *Consumer) HandleMsg(data <-chan amqp.Delivery) {
	for msg := range data {
		c.handler(msg.Body)
	}
}

func (p *Producer) Reset() {
	t := time.NewTimer(reconnectDelay)
	for {
		<-t.C
		if ch, err := p.Conn.Channel(); err != nil {
			log.Println("创建通道失败:", err)
			t.Reset(reconnectDelay)
		} else {
			p.Channel = ch
			err = ch.ExchangeDeclare(
				p.Config.Exchange, // name
				p.Config.Type,     // type
				p.Config.Durable,  // durable
				false,             // auto-deleted
				false,             // internal
				false,             // no-wait
				nil,               // arguments
			)
			if err != nil {
				log.Println("ExchangeDeclare error:", err)
				t.Reset(reconnectDelay)
				continue
			}
			return
		}
	}
}

func (p *Producer) PublishMsg(data interface{}) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = p.Channel.Publish(
		p.Config.Exchange,
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

func (p *Producer) PublishMsgWithKey(key string, data interface{}) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = p.Channel.Publish(
		p.Config.Exchange,
		key,
		false, //mandatory：true：如果exchange根据自身类型和消息routeKey无法找到一个符合条件的queue，那么会调用basic.return方法将消息返还给生产者。false：出现上述情形broker会直接将消息扔掉
		false, //如果exchange在将消息route到queue(s)时发现对应的queue上没有消费者，那么这条消息不会放入队列中。当与消息routeKey关联的所有queue(一个或多个)都没有消费者时，该消息会通过basic.return方法返还给生产者。
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        buf,
		})
	return err
}
