package Services

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	_ "github.com/mattn/go-sqlite3"
)

// NewMessageQueueProducer creates a new message queue service
func NewMessageQueueProducer() *kafka.Producer {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "172.172.147.132:30092",
		"client.id":         "Feeder-01",
		"acks":              "all"})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	// Test connectivity by getting metadata with a timeout
	metadata, err := p.GetMetadata(nil, false, 5000) // 5 second timeout
	if err != nil {
		fmt.Printf("Failed to connect to Kafka server: %s\n", err)
		p.Close()
		os.Exit(1)
	}

	// Verify we have at least one broker
	if len(metadata.Brokers) == 0 {
		fmt.Printf("No Kafka brokers found\n")
		p.Close()
		os.Exit(1)
	}

	fmt.Printf("Successfully connected to Kafka cluster with %d broker(s)\n", len(metadata.Brokers))
	return p
}

func NewMessageQueueConsumer(groupId *string) *kafka.Consumer {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "172.172.147.132:30092",
		"group.id":          *groupId,
		"auto.offset.reset": "earliest"})

	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	return c
}

func SendMessage(p *kafka.Producer, topic *string, value string) error {
	return p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: topic, Partition: kafka.PartitionAny},
		Value:          []byte(value)},
		nil, // delivery channel
	)
}

type DBService struct {
	Db       *sql.DB
	existing bool
}

func ConnectToDB() (*DBService, error) {
	var existing bool

	if _, err := os.Stat("gphotos.db"); os.IsNotExist(err) {
		existing = false
	}

	db, err := sql.Open("sqlite3", "gphotos.db")
	if err != nil {
		return nil, err
	}

	// Check if photos table exists and create it if it doesn't
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='photos'").Scan(&tableName)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		// Table doesn't exist, create it
		createTableSQL := `CREATE TABLE photos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL,
			checksum TEXT NOT NULL,
			date_added DATETIME DEFAULT CURRENT_TIMESTAMP
		)`

		_, err = db.Exec(createTableSQL)
		check(err)
	}

	return &DBService{db, existing}, nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
