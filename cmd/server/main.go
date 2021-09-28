package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	socketio "github.com/googollee/go-socket.io"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"

	"gitlab.com/kirbo/go-ruuvitag/internal/channels"
	"gitlab.com/kirbo/go-ruuvitag/internal/models"
)

var (
	rdb      *redis.Client
	rdbSlave *redis.Client
	server   *socketio.Server

	ctx            context.Context = context.Background()
	namespace      string          = "/"
	room           string          = ""
	updateEvent    string          = "update"
	initialEvent   string          = "initial"
	waitForSeconds time.Duration   = 5
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

	rdbSlave = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", "localhost", redisPort),
		DB:   0,
	})
}

func connectPostgres() {
	var (
		host     = os.Getenv("POSTGRES_HOST")
		port     = os.Getenv("POSTGRES_PORT")
		user     = os.Getenv("POSTGRES_USERNAME")
		password = os.Getenv("POSTGRES_PASSWORD")
		dbname   = os.Getenv("POSTGRES_DATABASE")
	)

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	dbConn, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}

	err = dbConn.Ping()
	if err != nil {
		panic(err)
	}

	log.Println("Successfully connected!")

	InitStore(&dbStore{db: dbConn})
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

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
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
		log.Printf("clientCount: %v    connected ID '%v'\n", clientCount, id)

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
		log.Printf("clientCount: %v    error %+v\n", clientCount, e)
	})

	server.OnDisconnect(namespace, func(s socketio.Conn, reason string) {
		id := s.ID()
		s.Close()
		clientCount := server.Count()
		log.Printf("clientCount: %v disconnected ID '%v'\n", clientCount, id)
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
	router.GET("/request-client-refresh", func(c *gin.Context) {
		var timestamp = makeTimestamp()
		err = rdbSlave.Publish(ctx, channels.Reload, timestamp).Err()
		if err != nil {
			log.Printf("request-client-refresh error: %s'\n", err)
		}

		broadcastClients("reload", "")
		c.String(http.StatusOK, "ok")
	})
	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))

	err = router.Run()
	if err != nil {
		panic(err)
	}
}

func subscribes() {
	var (
		devices = fmt.Sprintf("%s%s", channels.Device, "*")
		inserts = fmt.Sprintf("%s%s", channels.Insert, "*")
		reload = fmt.Sprintf("%s%s", channels.Reload, "*")
	)
	pubsub := rdbSlave.PSubscribe(ctx, devices, inserts, reload)

	_, err := pubsub.Receive(ctx)
	if err != nil {
		panic(err)
	}

	ch := pubsub.Channel()

	for msg := range ch {
		re := regexp.MustCompile(fmt.Sprintf(`^%s|%s|%s`, channels.Device, channels.Insert, channels.Reload))
		foundChannel := re.FindString(string(msg.Channel))
		switch foundChannel {
		case channels.Reload:
			go broadcastClients(msg.Payload)
		case channels.Device:
			go broadcastDevice(msg.Payload)
		case channels.Insert:
			go handleRow(msg.Channel, msg.Payload)
		default:
		}
	}
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

func broadcastClients(event string, message string) {
	server.BroadcastToRoom(namespace, room, event, message)
}

func broadcastDevice(row string) {
	device, err := parseMessage(row)
	if err != nil {
		panic(err)
	}

	broadcastMsg := broadcastMessage(device)

	server.BroadcastToRoom(namespace, room, updateEvent, broadcastMsg)
}

func handleRow(key, row string) {
	log.Printf("Handle key %s", key)

	device, err := parseMessage(row)
	if err != nil {
		panic(err)
	}

	store.InsertDevice(&device)
	go deleteKey(key, 1)
}

func deleteKey(key string, attempt int) {
	status := rdb.Del(ctx, key)
	if status.Val() == 1 {
		time.Sleep(waitForSeconds * time.Second)

		_, err := rdb.Get(ctx, key).Result()
		if err != nil {
			return
		}
	}

	if attempt == 5 {
		err := fmt.Errorf("Error deleting key %s status %s attempt #%v", key, status, attempt)
		log.Println("An error happened:", err)
		os.Exit(1)
	}

	attempt = attempt + 1
	log.Printf("Error deleting key %s, retrying in %v seconds", key, waitForSeconds)

	time.Sleep(waitForSeconds * time.Second)

	_, err := rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("key %s already deleted on master\n", key)
		return
	}

	log.Printf("attempt #%v for key %s\n", attempt, key)
	deleteKey(key, attempt)
}

func handleBuffer() {
	log.Println("Handle/clean buffer...")

	wg := sync.WaitGroup{}

	go func() {
		iter := rdbSlave.Scan(ctx, 0, fmt.Sprintf("%s%s", channels.Insert, "*"), 1).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			payload, err := rdbSlave.Get(ctx, key).Result()
			if err != nil {
				log.Printf("No data found for: %s", key)
				return
			}
			go handleRow(key, payload)
		}
		if err := iter.Err(); err != nil {
			panic(err)
		}
		wg.Done()
	}()
	wg.Add(1)
}

func initialDataForWebClient() (devices []models.BroadcastMessage, err error) {
	iter := rdbSlave.Scan(ctx, 0, fmt.Sprintf("%s%s", channels.Device, "*"), 1).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		row, err := rdbSlave.Get(ctx, key).Result()
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

func startTickers() {
	ticker := time.NewTicker(time.Duration(1) * time.Hour)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				handleBuffer()
			}
		}
	}()
}

func main() {
	connectRedis()
	connectPostgres()
	go startSocketIOServer()
	go handleBuffer()
	startTickers()
	subscribes()
}
