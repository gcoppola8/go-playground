package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	Services "github.com/gcoppola8/gphotos/services"
)

var metrics struct {
	folders int32
	files   int32
}

func main() {
	log.Println("Starting Feeder Service...")

	topic := "queue-gphotos"

	filePath := flag.String("path", "", "Path to the directory to process")
	//sqliteDB := flag.String("db", "", "Path to the sqlite database")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("File path is required. Use -path flag to specify the file path")
	}

	feeder := Services.NewMessageQueueProducer()

	log.Printf("Opening folder %s\n", *filePath)
	DigDir(filePath, feeder, &topic)

	log.Printf("Found %d files and %d folders\n", metrics.files, metrics.folders)

	// Flush any remaining messages and close producer
	feeder.Flush(1000) // Wait up to 1 seconds for messages to be delivered
	feeder.Close()

	log.Println("Feeder Service stopped.")
}

func DigDir(filePath *string, producer *kafka.Producer, topic *string) {

	absPath, err := filepath.Abs(*filePath)
	check(err)
	c, err := os.ReadDir(absPath)
	check(err)

	for _, entry := range c {
		if entry.IsDir() {
			metrics.folders += 1
			newPath := filepath.Join(absPath, entry.Name())
			DigDir(&newPath, producer, topic)
		} else {
			metrics.files += 1
			fullPath := filepath.Join(absPath, entry.Name())

			ext := filepath.Ext(fullPath)
			imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff", ".tif", ".heic", ".heif"}
			isImage := false
			for _, imgExt := range imageExts {
				if strings.EqualFold(ext, imgExt) {
					isImage = true
					break
				}
			}
			if !isImage {
				log.Printf("File: %s is not an image: SKIPPED", fullPath)
				continue
			}

			log.Printf("Found file %s\n", fullPath)

			if err := Services.SendMessage(producer, topic, fullPath); err != nil {
				log.Printf("Failed to send path to queue: %v", err)
			}
		}
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
