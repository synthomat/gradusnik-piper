package main

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	influxdb2 "github.com/influxdata/influxdb-client-go"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

type Aqara struct {
	Temperature float32 `json:"temperature"`
	Humidity float32 `json:"humidity"`
	Pressure float32 `json:"pressure"`
}

func main() {
	brokerUrl := "tcp://192.168.1.105:1883"
	log.Info("Connecting to broker ", brokerUrl)
	opts := mqtt.NewClientOptions().AddBroker(brokerUrl)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	client := influxdb2.NewClient("http://192.168.1.105:8086", "")
	// user blocking write client for writes to desired bucket
	writeAPI := client.WriteAPI("", "sensors")


	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := c.Subscribe("zigbee2mqtt/+", 0, func(client mqtt.Client, message mqtt.Message) {
		fmt.Println(message.Topic())
		var data Aqara
		if err := json.Unmarshal(message.Payload(), &data); err != nil {
			panic(err)
		}
		fmt.Println(data)
		p := influxdb2.NewPointWithMeasurement("sensor").
			AddTag("unit", "temperature").
			AddField("temp", data.Temperature).
			AddField("hum", data.Humidity).
			AddField("press", data.Pressure).
			SetTime(time.Now())
		writeAPI.WritePoint(p)

	}); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}



	select {}
}
