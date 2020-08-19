package main

import (
	"flag"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	influxdb2 "github.com/influxdata/influxdb-client-go"
	"github.com/influxdata/influxdb-client-go/api"
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strconv"
	p "synthomat.de/sensorius/piper"
	"time"
)

type MeasureData map[string]interface{}

type Storage interface {
	LogData(data p.Aqara)
}

type InfluxDB struct {
	client influxdb2.Client
	writer api.WriteAPI
}

func (idb InfluxDB) LogData(data p.Aqara) {
	fmt.Println("Logging ", data)

	tags := map[string]string{"unit": "temperature"}
	values := map[string]interface{}{
		"temp":  data.Temperature,
		"hum":   data.Humidity,
		"press": data.Pressure,
	}

	p := influxdb2.NewPoint(
		"sensor",
		tags,
		values,
		time.Now())

	idb.writer.WritePoint(p)
}

type InfluxDBConfig struct {
	Url   string
	Token string
}

func NewInfluxDBTarget(config InfluxDBConfig) (influx InfluxDB) {
	influx = InfluxDB{}
	influx.client = influxdb2.NewClient(config.Url, config.Token)
	influx.writer = influx.client.WriteAPI("", "sensors")

	return
}

type MQTTConfig struct {
	Topic  string
	Broker string
	User   string
	Pass   string
}

type Config struct {
	mqtt     MQTTConfig
	influxdb InfluxDBConfig
}

func GetConfig() Config {
	config := Config{}

	config.mqtt = MQTTConfig{
		Topic:  *flag.String("mqtt-topic", "sensors", "The topic name to/from which to publish/subscribe"),
		Broker: *flag.String("mqtt-broker", "tcp://192.168.1.105:1883", "The broker URI. ex: tcp://10.10.1.1:1883"),
		User:   *flag.String("mqtt-user", "", "The User (optional)"),
		Pass:   *flag.String("mqtt-password", "", "The password (optional)"),
	}

	config.influxdb = InfluxDBConfig{
		Url:   *flag.String("influx-url", "http://192.168.1.105:8086", "InfluxDB Server URL"),
		Token: *flag.String("influx-token", "", "InfluxDB Token (optional)"),
	}

	flag.Parse()

	return config
}

func main() {
	config := GetConfig()
	log.Info(config)

	target := NewInfluxDBTarget(config.influxdb)
	brokerUrl := config.mqtt.Broker

	log.Info("Connecting to broker ", brokerUrl)
	opts := mqtt.NewClientOptions().
		AddBroker(brokerUrl).
		SetUsername(config.mqtt.User).
		SetPassword(config.mqtt.Pass).
		SetKeepAlive(2 * time.Second).
		SetPingTimeout(1 * time.Second)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	messageHandler := func(client mqtt.Client, message mqtt.Message) {
		if val, err := strconv.ParseFloat(string(message.Payload()[:]), 32); err == nil {
			regp := *regexp.MustCompile(`rooms/(?P<room>.+)/(?P<metric>.+)`)
			grps := regp.FindAllStringSubmatch(message.Topic(), -1)
			var unit string
			switch grps[0][2] {
			case "humidity":
				unit = "percent"
			case "presure":
				unit = "millibar"
			case "temperature":
				unit = "celsius"
			}

			metric := fmt.Sprintf("%s_%s", grps[0][2], unit)
			tags := map[string]string{"room": grps[0][1]}

			values := map[string]interface{}{
				"value": val,
			}

			p := influxdb2.NewPoint(
				metric,
				tags,
				values,
				time.Now())

			target.writer.WritePoint(p)

		}
	}

	if token := c.Subscribe("rooms/#", 0, messageHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	select {}
}
