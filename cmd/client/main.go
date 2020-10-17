package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	socketio "github.com/googollee/go-socket.io"
	"github.com/imdario/mergo"
	_ "github.com/joho/godotenv/autoload"
	"github.com/patrickmn/go-cache"
	"github.com/peknur/ruuvitag"

	"github.com/kirbo/go-ruuvitag/internal/channels"
	"github.com/kirbo/go-ruuvitag/internal/models"
)

// Cache for variables
var (
	rdb    *redis.Client
	config models.JsonDevices
	server *socketio.Server
	client mqtt.Client

	mqttConfig   models.MQTTConfig = nil
	Cache        *cache.Cache      = cache.New(0, 0)
	ctx          context.Context   = context.Background()
	interval     time.Duration     = time.Minute
	namespace    string            = "/"
	room         string            = ""
	updateEvent  string            = "update"
	initialEvent string            = "initial"
)

func connectRedis() {
	var (
		redisPassword = os.Getenv("REDIS_MASTER_PASSWORD")
		redisHost     = os.Getenv("REDIS_MASTER_HOST")
		redisPort     = os.Getenv("REDIS_MASTER_PORT")
	)

	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0,
	})
}

func connectMQTT() {
	if mqttConfig != nil {
		uriString := fmt.Sprintf("tcp://%s:%s@%s:%v", mqttConfig.User.Username, mqttConfig.User.Password, mqttConfig.Host, mqttConfig.Port)
		fmt.Printf("uriString: %s\n", uriString)

		uri, err := url.Parse(uriString)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("uri: %+v\n", uri)

		client = connect("pub", uri)
	}
}

func connect(clientId string, uri *url.URL) mqtt.Client {
	opts := createClientOptions(clientId, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()

	for !token.WaitTimeout(3 * time.Second) {
	}

	if err := token.Error(); err != nil {
		log.Fatal(err)
	}

	return client
}

func createClientOptions(clientId string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientId)

	return opts
}

func loadMQTTConfigs() {
	if _, err := os.Stat("./mqtt.json"); err == nil {
		jsonFile, err := os.Open("./mqtt.json")
		if err != nil {
			fmt.Print(err)
		}

		byteValue, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(byteValue, &mqttConfig)
	}
}

func loadConfigs() {
	log.Print("Reloading configs...")
	jsonFile, err := os.Open("./config.json")
	if err != nil {
		fmt.Print(err)
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &config)

	for i := 0; i < len(config); i++ {
		Cache.Set(fmt.Sprintf("name:%s", config[i].ID), config[i].Name, cache.NoExpiration)
		if x, found := Cache.Get(fmt.Sprintf("%s%s", channels.Device, config[i].ID)); found {
			device := x.(models.Device)
			var (
				ping        float32 = float32(device.Ping) / 1000
				name                = device.Name
				temperature         = device.Temperature
				humidity            = device.Humidity
				pressure    float32 = float32(device.Pressure) / 100
				battery             = device.Battery
			)

			fmt.Println(fmt.Sprintf("%9.3fs ago - %-14s :: %7.2f °c, %6.2f %%H, %7.2f hPa, %5.3f v", ping, name, temperature, humidity, pressure, battery))
		}
	}
}

func startTickers() {
	ticker := time.NewTicker(interval)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				loadConfigs()
				createInserts()
			}
		}
	}()
}

func GinMiddleware(allowOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Content-Length, X-CSRF-Token, Token, session, Origin, Host, Connection, Accept-Encoding, Accept-Language, X-Requested-With")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Request.Header.Del("Origin")

		c.Next()
	}
}

func startSocketIOServer() {
	var err error

	router := gin.New()

	server, err = socketio.NewServer(nil)
	if err != nil {
		panic(err)
	}

	server.OnConnect(namespace, func(s socketio.Conn) error {
		id := s.ID()
		s.Join(room)
		clientCount := server.Count()
		fmt.Printf("clientCount: %v    connected ID '%v'\n", clientCount, id)

		devices, err := initialDataForWebClient()
		if err != nil {
			panic(err)
		}

		server.BroadcastToRoom(namespace, room, initialEvent, devices)

		return nil
	})

	server.OnError(namespace, func(s socketio.Conn, e error) {
		s.Close()
		clientCount := server.Count()
		fmt.Printf("clientCount: %v    error %+v\n", clientCount, e)
	})

	server.OnDisconnect(namespace, func(s socketio.Conn, reason string) {
		id := s.ID()
		s.Close()
		clientCount := server.Count()
		fmt.Printf("clientCount: %v disconnected ID '%v'\n", clientCount, id)
	})

	wg := sync.WaitGroup{}

	go func() {
		err = server.Serve()
		if err != nil {
			panic(err)
		}
		wg.Done()
	}()
	wg.Add(1)

	defer server.Close()

	router.Use(GinMiddleware("*"))
	router.Use(static.Serve("/", static.LocalFile("./floorplan/dist", true)))
	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))

	err = router.Run()
	if err != nil {
		panic(err)
	}
}

func createInserts() {
	for i := range config {
		sensor := &config[i]

		oldID := parseOldID(sensor.ID)

		val, err := rdb.Get(ctx, fmt.Sprintf("%s%s", channels.Device, oldID)).Result()
		if err != nil {
			log.Printf("No data found for: %s", sensor.Name)
			return
		}

		if err = setAndPublish(fmt.Sprintf("%s%v:%s", channels.Insert, makeTimestamp(), oldID), val); err != nil {
			panic(err)
		}
	}
}

func setAndPublish(channel string, data string) error {
	var err error

	err = rdb.Publish(ctx, channel, data).Err()
	if err != nil {
		return err
	}

	err = rdb.Set(ctx, channel, data, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func parseOldID(id string) string {
	return strings.ToLower(strings.Replace(id, ":", "", -1))
}

func parseMessage(row string) (device models.Device, err error) {
	err = json.Unmarshal([]byte(row), &device)
	if err != nil {
		return
	}

	return
}

func broadcastMessage(device models.Device) models.BroadcastMessage {
	return models.BroadcastMessage{
		Timestamp:   device.TimestampZ,
		TagID:       device.OldID,
		Name:        device.Name,
		Temperature: device.Temperature,
		Pressure:    device.Pressure,
		Ping:        device.Ping,
		Battery:     device.Battery,
		Humidity:    device.Humidity,
	}
}

func broadcastDevice(row string) {
	device, err := parseMessage(row)
	if err != nil {
		panic(err)
	}

	broadcastMsg := broadcastMessage(device)

	server.BroadcastToRoom(namespace, room, updateEvent, broadcastMsg)
}

func broadcastMQTTDevice(device models.Device) {
	if client != nil {
		var (
			battery      = device.Battery
			humidity     = device.Humidity
			pressure     = device.Pressure
			temperature  = device.Temperature
			acceleration = device.Acceleration

			topic = "ruuvitag/" + device.ID + "/"

			topicA = topic + "acceleration"
			topicB = topic + "battery"
			topicH = topic + "humidity"
			topicP = topic + "pressure"
			topicT = topic + "temperature"
		)

		client.Publish(topicA, 0, true, acceleration)
		client.Publish(topicB, 0, true, battery)
		client.Publish(topicH, 0, true, humidity)
		client.Publish(topicP, 0, true, pressure)
		client.Publish(topicT, 0, true, temperature)
	}
}

func initialDataForWebClient() (devices []models.BroadcastMessage, err error) {
	iter := rdb.Scan(ctx, 0, fmt.Sprintf("%s%s", channels.Device, "*"), 1).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		row, err := rdb.Get(ctx, key).Result()
		if err != nil {
			log.Printf("No data found for: %s", key)
			panic(err)
		}

		device, err := parseMessage(row)
		if err != nil {
			panic(err)
		}

		devices = append(devices, broadcastMessage(device))
	}
	if err := iter.Err(); err != nil {
		panic(err)
	}

	return
}

func handler(data ruuvitag.Measurement) {
	var (
		address    = data.DeviceID()
		addressOld = parseOldID(address)
		ping       = int64(0)
		name       = ""
	)

	timestamp := makeTimestamp()
	if value, found := Cache.Get(fmt.Sprintf("lastTimestamp:%s", address)); found {
		var lastTimestamp = value.(int64)
		ping = timestamp - lastTimestamp
	}

	if value, found := Cache.Get(fmt.Sprintf("name:%s", address)); found {
		name = value.(string)
	}

	var device = models.Device{
		ID:    address,
		OldID: addressOld,
		Name:  name,
	}

	var deviceStub = models.Device{
		Ping:        ping,
		Format:      data.Format(),
		Humidity:    data.Humidity(),
		Temperature: data.Temperature(),
		Pressure:    float32(data.Pressure()) / float32(100),
		Timestamp:   timestamp,
		TimestampZ:  time.Unix(int64(time.Duration(timestamp/1000)), 0).Format(time.RFC3339),
		Acceleration: models.DeviceAcceleration{
			X: data.AccelerationX(),
			Y: data.AccelerationY(),
			Z: data.AccelerationZ(),
		},
		Battery: data.BatteryVoltage(),
	}

	if err := mergo.Merge(&device, deviceStub); err != nil {
		panic(err)
	}

	Cache.Set(fmt.Sprintf("%s%s", channels.Device, address), device, cache.NoExpiration)
	Cache.Set(fmt.Sprintf("lastTimestamp:%s", address), timestamp, cache.NoExpiration)

	// log.Printf("%s[v%d] %s : %+v", address, data.Format(), name, deviceStub)

	redisData, err := json.Marshal(device)
	if err != nil {
		panic(err)
	}

	broadcastDevice(string(redisData))
	broadcastMQTTDevice(device)

	if err = setAndPublish(fmt.Sprintf("%s%s", channels.Device, addressOld), string(redisData)); err != nil {
		panic(err)
	}
}

func main() {
	connectRedis()
	go startSocketIOServer()
	loadConfigs()
	startTickers()

	scanner, err := ruuvitag.OpenScanner(10)
	if err != nil {
		log.Fatal(err)
	}

	output := scanner.Start()
	for {
		data := <-output
		go handler(data)
	}
}
