package main

import (
	"errors"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	m, err := migrate.New(
		"file://./migrations",
		databaseURL,
	)
	if err != nil {
		log.Fatalf("create migration instance: %v", err)
	}

	defer func() {
		sourceErr, databaseErr := m.Close()

		if sourceErr != nil {
			log.Printf("migration source close error: %v", sourceErr)
		}

		if databaseErr != nil {
			log.Printf("migration database close error: %v", databaseErr)
		}
	}()

	err = m.Up()

	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("no new migrations")
		return
	}

	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		log.Fatalf("get migration version: %v", err)
	}

	log.Printf("migration completed: version=%d dirty=%v", version, dirty)
}
