package main

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	p "synthomat.de/sensorius/piper"
	"time"
)

var sensorToRoom = map[string]string{
	"ws-aqara-01": "bedroom",
	"ws-aqara-02": "livingroom",
	"ws-aqara-03": "office",
	"ws-aqara-04": "childroom",
}

func main() {
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://192.168.1.105:1883").
		SetUsername("").
		SetPassword("").
		SetKeepAlive(2 * time.Second).
		SetPingTimeout(1 * time.Second)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	messageHandler := func(client mqtt.Client, message mqtt.Message) {
		topic := message.Topic()
		sensor := strings.TrimPrefix(topic, "zigbee2mqtt/")

		if room, ok := sensorToRoom[sensor]; ok {
			var data p.Aqara
			if err := json.Unmarshal(message.Payload(), &data); err != nil {
				log.Error("Cannot parse JSON", err)
				return
			}

			client.Publish(fmt.Sprintf("rooms/%s/humidity", room), 0, false, fmt.Sprintf("%.3f", data.Humidity/100)).Wait()
			client.Publish(fmt.Sprintf("rooms/%s/presure", room), 0, false, fmt.Sprintf("%.1f", data.Pressure)).Wait()
			client.Publish(fmt.Sprintf("rooms/%s/temperature", room), 0, false, fmt.Sprintf("%.1f", data.Temperature)).Wait()
		}
	}

	if token := c.Subscribe("zigbee2mqtt/+", 0, messageHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	select {}
}
