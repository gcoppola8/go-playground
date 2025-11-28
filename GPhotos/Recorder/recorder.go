package main

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	Services "github.com/gcoppola8/gphotos/services"
)

func main() {
	log.Println("Starting Recorder Service...")

	topic := "queue-gphotos"
	nextTopic := "ids"

	recorderID := "Recorder-01"
	recorder := Services.NewMessageQueueConsumer(&recorderID)

	dbService, err := Services.ConnectToDB()
	check(err)

	err = recorder.SubscribeTopics([]string{topic}, nil)
	check(err)

	defer recorder.Close()

	producer := Services.NewMessageQueueProducer()
	defer producer.Close()

	for {
		ev := recorder.Poll(2000)
		switch e := ev.(type) {
		case *kafka.Message:

			path := string(e.Value)
			checksum := CalculateChecksumPhotoAt(path)

			var existingPhoto struct {
				ID        int
				Path      string
				Checksum  string
				DateAdded string
			}

			row := dbService.Db.QueryRow("SELECT id, path, checksum, date_added FROM photos WHERE path = ? OR checksum = ?", path, checksum)
			err = row.Scan(&existingPhoto.ID, &existingPhoto.Path, &existingPhoto.Checksum, &existingPhoto.DateAdded)
			if err != nil && err != os.ErrNotExist && err.Error() != "sql: no rows in result set" {
				log.Printf("Error querying database: %v", err)
			}

			if err == sql.ErrNoRows {
				// No existing photo found - insert new one
				result, err := dbService.Db.Exec("INSERT INTO photos (path, checksum, date_added) VALUES (?, ?, unixepoch())", path, checksum)
				if err != nil {
					log.Printf("Error inserting into database: %v", err)
				} else {
					id, err := result.LastInsertId()
					if err != nil {
						log.Printf("Error getting last insert ID: %v", err)
						log.Printf("Inserted new photo: %s (ID: %d)", path, id)
					}

					Services.SendMessage(producer, &nextTopic, strconv.FormatInt(id, 10))
				}
			} else if err != nil {
				log.Printf("Error querying database: %v", err)
			} else {
				// Photo already exists
				log.Printf("Photo already exists (ID: %d, Path: %s)", existingPhoto.ID, existingPhoto.Path)
			}

		case kafka.Error:
			fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
		default:
			fmt.Printf("Ignored %v\n", e)
		}
	}

}

func CalculateChecksumPhotoAt(path string) string {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return ""
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		log.Printf("Error calculating checksum: %v", err)
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
