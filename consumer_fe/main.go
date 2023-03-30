package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/streadway/amqp"
	"github.com/xor-shift/teleserver/common"
	"log"
	"net/http"
	"os"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("loading dotenv failed: %s", err)
	}
}

func main() {
	var err error

	var consumer *common.AMQPConsumer
	var app *iris.Application

	var lastFullPacket common.AMQPPacket

	if consumer, err = common.NewAMQPConsumer(
		"consumer_fe_queue",
		"consumer_fe_consumer",
		func(delivery amqp.Delivery) error {
			var amqpPacket common.AMQPPacket

			if amqpPacket, err = common.ParseAMQPPacket(&delivery); err != nil {
				log.Printf("error decoding a packet with gob: %s", err)
			}

			packet := amqpPacket.Packet
			fullPacket, ok := packet.Inner.(common.FullPacket)
			if !ok {
				return nil
			}

			log.Printf("%d @ %d: %d, %d (%d/%d)",
				packet.SequenceID,
				packet.Timestamp,
				fullPacket.QueueFillAmount,
				fullPacket.FreeHeap,
				fullPacket.AllocCount,
				fullPacket.FreeCount)

			lastFullPacket = amqpPacket

			return nil
		}); err != nil {
		log.Fatalln(err)
	}

	if err = consumer.Start(); err != nil {
		log.Fatalln(err)
	}

	app = iris.New()

	app.Get("/test", func(ctx iris.Context) {
		_, _ = ctx.Text("OK")
	})

	app.Get("/data", func(ctx iris.Context) {
		var jsonData []byte
		jsonData, err = json.Marshal(lastFullPacket)

		if err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			ctx.Text("internal error: %s", err)
			return
		}

		ctx.ContentType("application/json")
		_, _ = ctx.Text(string(jsonData))
	})

	if err = app.Listen(fmt.Sprintf(":%s", os.Getenv("CONSUMER_FE_PORT"))); err != nil {
		log.Fatalln(err)
	}
}
