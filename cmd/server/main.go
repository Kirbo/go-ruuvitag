package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"

	"github.com/kirbo/go-ruuvitag/internal/channels"
	"github.com/kirbo/go-ruuvitag/internal/models"
)

var (
	rdb      *redis.Client
	rdbSlave *redis.Client
	ctx      = context.Background()
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

	fmt.Println("Successfully connected!")

	InitStore(&dbStore{db: dbConn})
}

func subscribes() {
	var (
		devices = fmt.Sprintf("%s%s", channels.Device, "*")
		inserts = fmt.Sprintf("%s%s", channels.Insert, "*")
	)
	pubsub := rdbSlave.PSubscribe(ctx, devices, inserts)

	_, err := pubsub.Receive(ctx)
	if err != nil {
		panic(err)
	}

	ch := pubsub.Channel()

	for msg := range ch {
		re := regexp.MustCompile(fmt.Sprintf(`^%s|%s`, channels.Device, channels.Insert))
		foundChannel := re.FindString(string(msg.Channel))
		switch foundChannel {
		// case channels.Device:
		// log.Println("Device", msg.Payload)
		case channels.Insert:
			handleRow(msg.Channel, msg.Payload)
		default:
			// log.Println("Unknown channel", msg.Channel, "foundChannel", foundChannel)
		}
	}
}

func handleRow(key, row string) {
	log.Printf("Handle key %s", key)

	var device models.Device
	err := json.Unmarshal([]byte(row), &device)
	if err != nil {
		panic(err)
	}

	store.InsertDevice(&device)
	deleteKey(key)
}

func deleteKey(key string) {
	status := rdb.Del(ctx, key)
	log.Printf("key %s status %s", key, status)
	if status.Val() == 1 {
		return
	}

	for {
		log.Printf("key %s retry in second", key)
		time.Sleep(time.Second)
		status = rdb.Del(ctx, key)
		log.Printf("key %s status %s", key, status)
		if status.Val() == 1 {
			break
		}
	}
}

func handleBuffer() {
	iter := rdbSlave.Scan(ctx, 0, fmt.Sprintf("%s%s", channels.Insert, "*"), 1).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		payload, err := rdbSlave.Get(ctx, key).Result()
		if err != nil {
			log.Printf("No data found for: %s", key)
			return
		}
		handleRow(key, payload)
	}
	if err := iter.Err(); err != nil {
		panic(err)
	}
}

func main() {
	connectRedis()
	connectPostgres()
	handleBuffer()
	subscribes()
}
