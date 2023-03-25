package common

import (
	"bytes"
	"encoding/gob"
	"github.com/streadway/amqp"
	"log"
	"os"
	"sync"
)

type AMQPConsumer struct {
	amqpConn  *amqp.Connection
	amqpChan  *amqp.Channel
	amqpQueue amqp.Queue

	queueName    string
	consumerName string

	amqpConsumer <-chan amqp.Delivery
	callback     func(amqp.Delivery) error
	wg           sync.WaitGroup
}

func NewAMQPConsumer(queueName, consumerName string, callback func(amqp.Delivery) error) (*AMQPConsumer, error) {
	var err error
	consumer := AMQPConsumer{
		callback: callback,

		queueName:    queueName,
		consumerName: consumerName,
	}

	if consumer.amqpConn, err = amqp.Dial(os.Getenv("AMQP_URL")); err != nil {
		return nil, err
	}

	if consumer.amqpChan, err = consumer.amqpConn.Channel(); err != nil {
		_ = consumer.amqpConn.Close()
		return nil, err
	}

	if err = consumer.amqpChan.ExchangeDeclare(
		"full_packets", // name
		"fanout",       // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	); err != nil {
		_ = consumer.amqpChan.Close()
		_ = consumer.amqpConn.Close()
		return nil, err
	}

	if consumer.amqpQueue, err = consumer.amqpChan.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		true,      // exclusive
		false,     // no-wait
		nil,       // arguments
	); err != nil {
		_ = consumer.amqpChan.Close()
		_ = consumer.amqpConn.Close()
		return nil, err
	}

	if err = consumer.amqpChan.QueueBind(
		consumer.amqpQueue.Name, // queue name
		"",                      // routing key
		"full_packets",          // exchange
		false,
		nil,
	); err != nil {
		_ = consumer.amqpChan.Close()
		_ = consumer.amqpConn.Close()
		return nil, err
	}

	return &consumer, nil
}

func (c *AMQPConsumer) Start() error {
	var err error

	if c.amqpConsumer, err = c.amqpChan.Consume(
		c.amqpQueue.Name, // queue
		c.consumerName,   // consumer
		true,             // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
	); err != nil {
		return err
	}

	c.wg.Add(1)

	go func() {
		defer func() { c.wg.Done() }()

		for {
			delivery, open := <-c.amqpConsumer
			if !open {
				break
			}

			c.callback(delivery)
		}
	}()

	return nil
}

func (c *AMQPConsumer) Stop() error {
	var err error

	if err = c.amqpChan.Cancel(c.consumerName, false); err != nil {
		return err
	}

	return nil
}

func (c *AMQPConsumer) Wait() {
	c.wg.Wait()
}

func (c *AMQPConsumer) Close() error {
	var err error

	if err = c.amqpChan.Close(); err != nil {
		return err
	}

	if err = c.amqpConn.Close(); err != nil {
		return err
	}

	return nil
}

func ParseAMQPPacket(delivery *amqp.Delivery) (AMQPPacket, error) {
	var err error
	var amqpPacket AMQPPacket

	buffer := bytes.NewBuffer(delivery.Body)
	decoder := gob.NewDecoder(buffer)

	if err = decoder.Decode(&amqpPacket); err != nil {
		log.Printf("error decoding a packet with gob: %s", err)
	}

	return amqpPacket, nil
}
