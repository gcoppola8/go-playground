module Feeder

go 1.24

require github.com/gcoppola8/gphotos/services v0.0.0

require (
	github.com/confluentinc/confluent-kafka-go v1.9.2 // indirect
	github.com/mattn/go-sqlite3 v1.14.32 // indirect
)

replace github.com/gcoppola8/gphotos/services => ../services
