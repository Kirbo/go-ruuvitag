package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	socketio "github.com/googollee/go-socket.io"
	_ "github.com/joho/godotenv/autoload"
	"github.com/peknur/ruuvitag"

	"gitlab.com/kirbo/go-ruuvitag/internal/channels"
	"gitlab.com/kirbo/go-ruuvitag/internal/models"
)

var (
	rdb        *redis.Client
	config     models.Config
	server     *socketio.Server
	mqttClient mqtt.Client
	mqttConfig models.MQTTConfig

	ctx          context.Context = context.Background()
	namespace    string          = "/"
	room         string          = ""
	updateEvent  string          = "update"
	initialEvent string          = "initial"
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

	reloadNames()
}

func connectMQTT() {
	if _, err := os.Stat("./mqtt.json"); err == nil {
		jsonFile, err := os.Open("./mqtt.json")
		if err != nil {
			fmt.Print(err)
		}

		byteValue, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(byteValue, &mqttConfig)

		uriString := fmt.Sprintf("tcp://%s:%s@%s:%v", mqttConfig.User.Username, mqttConfig.User.Password, mqttConfig.Host, mqttConfig.Port)
		fmt.Printf("uriString: %s\n", uriString)

		uri, err := url.Parse(uriString)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("uri: %+v\n", uri)

		opts := createClientOptions(mqttConfig.User.CliendID, uri)
		mqttClient = mqtt.NewClient(opts)
		token := mqttClient.Connect()

		for !token.WaitTimeout(3 * time.Second) {
		}

		if err := token.Error(); err != nil {
			log.Fatal(err)
		}
	}
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

func loadConfigs() {
	log.Print("Reloading configs...")
	jsonFile, err := os.Open("./config.json")
	if err != nil {
		fmt.Print(err)
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &config)
}

func reloadNames() {
	if config.LogReloadNames {
		log.Print("Reloading sensor names...")
	}

	for i := 0; i < len(config.Ruuvitags); i++ {
		sensor := &config.Ruuvitags[i]
		oldID := parseOldID(sensor.ID)
		key := fmt.Sprintf("%s%s", channels.Device, oldID)

		var device models.Device
		var found bool = false

		val, err := rdb.Get(ctx, key).Result()
		if err == nil {
			found = true
			device, err = parseMessage(val)
			if err != nil {
				return
			}
		}

		device.Name = sensor.Name

		redisData, err := stringifyMessage(device)
		if err != nil {
			return
		}

		err = rdb.Set(ctx, key, redisData, 0).Err()
		if err != nil {
			return
		}

		var timestamp = makeTimestamp()
		var ping float32 = float32(timestamp-device.Timestamp) / 1000

		if found {
			fmt.Println(fmt.Sprintf("%9.3fs ago - %-14s :: %7.2f °c, %6.2f %%H, %7.2f hPa, %5.3f v", ping, device.Name, device.Temperature, device.Humidity, device.Pressure, device.Battery))
		}
	}
}

func startTickers() {
	configTicker := time.NewTicker(time.Minute)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-configTicker.C:
				loadConfigs()
				reloadNames()
			}
		}
	}()

	if config.EnableInserts {
		insertsTicker := time.NewTicker(time.Duration(config.Interval) * time.Second)
		done := make(chan bool)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-insertsTicker.C:
					createInserts()
				}
			}
		}()
	}

	if config.EnableMQTT {
		mqttTicker := time.NewTicker(time.Duration(mqttConfig.Interval) * time.Second)
		done := make(chan bool)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-mqttTicker.C:
					broadcastMQTTDevices()
				}
			}
		}()
	}

	if config.EnableSocket && config.PushSocketOnIntervals {
		mqttTicker := time.NewTicker(time.Duration(config.SocketInterval) * time.Second)
		done := make(chan bool)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-mqttTicker.C:
					broadcastSocketDevices()
				}
			}
		}()
	}
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
		fmt.Printf("Ennen serve\n")
		err = server.Serve()
		fmt.Printf("Serve jälkeen\n")
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
	if config.LogInserts {
		log.Print("Inserting to Database...")
	}

	for i := range config.Ruuvitags {
		sensor := &config.Ruuvitags[i]

		oldID := parseOldID(sensor.ID)

		val, err := rdb.Get(ctx, fmt.Sprintf("%s%s", channels.Device, oldID)).Result()
		if err != nil {
			log.Printf("No data found for: %s", sensor.Name)
			return
		}

		if err = setNXAndPublish(fmt.Sprintf("%s%v:%s", channels.Insert, makeTimestamp(), oldID), val); err != nil {
			panic(err)
		}
	}
}

func setNXAndPublish(channel string, data string) error {
	var err error

	err = rdb.Publish(ctx, channel, data).Err()
	if err != nil {
		return err
	}

	err = rdb.SetNX(ctx, channel, data, 0).Err()
	if err != nil {
		return err
	}

	return nil
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

func stringifyMessage(device models.Device) (stringified string, err error) {
	var marhalled []byte
	marhalled, err = json.Marshal(device)
	if err != nil {
		return
	}

	stringified = string(marhalled)

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

func broadcastClients(event string, message string) {
	if config.EnableSocket {
		log.Print(fmt.Sprintf("%s - %s", event, message))
		server.BroadcastToRoom(namespace, room, event, message)
	}
}

func broadcastDevice(row string) {
	if config.LogSocket && config.PushSocketImmediatelly {
		log.Print("Broadcasting to Socket.IO...")
	}

	device, err := parseMessage(row)
	if err != nil {
		panic(err)
	}

	broadcastMsg := broadcastMessage(device)

	server.BroadcastToRoom(namespace, room, updateEvent, broadcastMsg)
}

func broadcastMQTTDevices() {
	if config.LogMQTT {
		log.Print("Publishing to MQTT...")
	}

	for i := range config.Ruuvitags {
		sensor := &config.Ruuvitags[i]

		timestamp := makeTimestamp()
		oldID := parseOldID(sensor.ID)

		val, err := rdb.Get(ctx, fmt.Sprintf("%s%s", channels.Device, oldID)).Result()
		if err != nil {
			log.Printf("No data found for: %s", sensor.Name)
			return
		}

		device, err := parseMessage(val)
		if err != nil {
			panic(err)
		}

		device.Ping = int64(timestamp - device.Timestamp)

		go broadcastMQTTDevice(device)
	}
}

func broadcastSocketDevices() {
	if config.LogSocket {
		log.Print("Publishing to Socket.io...")
	}

	for i := range config.Ruuvitags {
		sensor := &config.Ruuvitags[i]

		timestamp := makeTimestamp()
		oldID := parseOldID(sensor.ID)

		val, err := rdb.Get(ctx, fmt.Sprintf("%s%s", channels.Device, oldID)).Result()
		if err != nil {
			log.Printf("No data found for: %s", sensor.Name)
			return
		}

		device, err := parseMessage(val)
		if err != nil {
			panic(err)
		}

		device.Ping = int64(timestamp - device.Timestamp)

		redisData, err := stringifyMessage(device)
		if err != nil {
			panic(err)
		}

		go broadcastDevice(redisData)
	}
}

func broadcastMQTTDevice(device models.Device) {
	topic := fmt.Sprintf("ruuvitag/%v", device.ID)

	broadcastMsg, err := json.Marshal(device)
	if err != nil {
		panic(err)
	}

	token := mqttClient.Publish(topic, 0, mqttConfig.RetainMessages, broadcastMsg)
	token.Wait()
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
		timestamp  = makeTimestamp()
		address    = data.DeviceID()
		addressOld = parseOldID(address)
		ping       = int64(0)
		key        = fmt.Sprintf("%s%s", channels.Device, addressOld)
	)

	var device = models.Device{
		ID:    address,
		OldID: addressOld,
	}

	row, err := rdb.Get(ctx, key).Result()
	if err == nil {
		device, err = parseMessage(row)
		if err != nil {
			panic(err)
		}

		ping = int64(timestamp - device.Timestamp)
	}

	device = models.Device{
		ID:          address,
		OldID:       addressOld,
		Name:        device.Name,
		Ping:        ping,
		Format:      data.Format(),
		Humidity:    data.Humidity(),
		Temperature: data.Temperature(),
		Battery:     data.BatteryVoltage(),
		Pressure:    float32(data.Pressure()) / float32(100),
		Timestamp:   timestamp,
		TimestampZ:  time.Unix(int64(time.Duration(timestamp/1000)), 0).Format(time.RFC3339),
		Acceleration: models.DeviceAcceleration{
			X: data.AccelerationX(),
			Y: data.AccelerationY(),
			Z: data.AccelerationZ(),
		},
	}

	redisData, err := stringifyMessage(device)
	if err != nil {
		panic(err)
	}

	if config.EnableSocket && config.PushSocketImmediatelly {
		go broadcastDevice(redisData)
	}

	fmt.Sprintf("redisData: %+v", redisData)

	if err = setAndPublish(key, redisData); err != nil {
		panic(err)
	}
}

func subscribes() {
	log.Print("Subscribe to channels...")
	var reload = fmt.Sprintf("%s%s", channels.Reload, "*")
	pubsub := rdb.PSubscribe(ctx, reload)

	_, err := pubsub.Receive(ctx)
	if err != nil {
		panic(err)
	}

	ch := pubsub.Channel()

	for msg := range ch {
		re := regexp.MustCompile(fmt.Sprintf(`^%s`, channels.Reload))
		foundChannel := re.FindString(string(msg.Channel))
		switch foundChannel {
		case channels.Reload:
			go broadcastClients(msg.Channel, msg.Payload)
		default:
		}
	}
}

func startScanning() {
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

func main() {
	loadConfigs()

	if config.EnableRedis {
		connectRedis()
		go subscribes()
	}

	if config.EnableMQTT {
		connectMQTT()
	}

	if config.EnableSocket {
		go startSocketIOServer()
	}

	startTickers()
	startScanning()
}
