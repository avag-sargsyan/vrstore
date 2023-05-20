package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

const (
	promotionsCSVFile = "/app/promotions/promotions.csv"
	chunkSize         = 1000
)

// Promotion represents a promotion record
type Promotion struct {
	ID             string    `json:"id"`
	Price          float64   `json:"price"`
	ExpirationDate time.Time `json:"expiration_date"`
}

var db *sql.DB

func main() {
	//Connect to the PostgreSQL database
	db = connectDB()
	defer db.Close()

	// Perform database migrations
	err := doMigration()
	if err != nil {
		log.Fatal("Error performing database migrations:", err)
	}

	// Run a routine every 30 minutes to process the CSV file
	go processCSVFile()

	// Set up the HTTP endpoint
	http.HandleFunc("/promotions/", getPromotionByID)
	log.Fatal(http.ListenAndServe(":1321", nil))
}

// connectDB establishes a connection to the PostgreSQL database
func connectDB() *sql.DB {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DATABASE_HOST"), os.Getenv("DATABASE_PORT"), os.Getenv("DATABASE_USER"),
		os.Getenv("DATABASE_PASSWORD"), os.Getenv("DATABASE_NAME"))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Make sure connected to the database
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return db
}

// doMigration performs database migrations
func doMigration() error {
	// Run the SQL migration scripts
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS promotions (
			id uuid PRIMARY KEY,
			price NUMERIC(12, 6) NOT NULL,
  			expiration_date TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

// processCSVFile reads the CSV file and inserts the records into the database
func processCSVFile() {
	for {
		//Clean up the storage in the first place
		err := cleanUpStorage()
		if err != nil {
			log.Println("Error deleting existing records:", err)
			continue
		}

		// Process promotions
		file, err := os.Open(promotionsCSVFile)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		reader.Comma = ','

		// Read the CSV records in chunks in case the file is too big
		chunk := make([][]string, 0, chunkSize)

		var wg sync.WaitGroup

		for {
			// Read a record from the CSV file
			record, err := reader.Read()
			if err != nil {
				// Handle end of file or any error
				if err.Error() == "EOF" {
					break
				}
				log.Fatal(err)
			}

			// Add the record to the current chunk
			chunk = append(chunk, record)

			// Process the chunk if it reaches the desired size
			if len(chunk) == chunkSize {
				wg.Add(1)
				// Insert the chunk into the database concurrently
				go insertPromotions(chunk, &wg)
				chunk = make([][]string, 0, chunkSize)
			}
		}

		// Process the last chunk
		if len(chunk) > 0 {
			wg.Add(1)
			go insertPromotions(chunk, &wg)
		}

		// Wait for all goroutines to finish
		wg.Wait()

		fmt.Println("Processing complete")

		// Sleep for 30 minutes before the next file processing
		time.Sleep(30 * time.Minute)
	}
}

// cleanUpStorage deletes all existing records in the database
func cleanUpStorage() error {
	_, err := db.Exec("DELETE FROM promotions")
	return err
}

// insertPromotions inserts bulk records into the database
func insertPromotions(records [][]string, wg *sync.WaitGroup) error {
	txn, err := db.Begin()
	if err != nil {
		log.Println("Error starting a transaction", err)
	}

	stmt, err := txn.Prepare(pq.CopyIn("promotions", "id", "price", "expiration_date"))
	if err != nil {
		log.Println(err)
	}

	for _, record := range records {
		id := record[0]
		price := record[1]
		expirationDate := record[2]

		// Parse the price and expiration date
		priceFloat, err := strconv.ParseFloat(price, 64)
		if err != nil {
			log.Println("Error parsing price:", err)
			continue
		}

		// Parse the expiration date
		expirationDateTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", expirationDate)
		if err != nil {
			log.Println("Error parsing expiration date:", err)
			continue
		}

		_, err = stmt.Exec(id, priceFloat, expirationDateTime)
		if err != nil {
			log.Println(err)
			continue
		}
	}

	// Flush the records to the database
	_, err = stmt.Exec()
	if err != nil {
		log.Println("Error inserting promotions into database:", err)
	}

	// Commit the transaction
	err = txn.Commit()
	if err != nil {
		log.Println("Error committing the transaction:", err)
	}

	// Close the prepared statement
	err = stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	wg.Done()

	return nil
}

// getPromotionByID returns a promotion by ID
func getPromotionByID(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from the request URL
	id := r.URL.Path[len("/promotions/"):]

	// Query the database for the promotion by ID
	query := "SELECT id, price, expiration_date FROM promotions WHERE id = $1"
	row := db.QueryRow(query, id)

	// Read the promotion values from the database row
	var promotion Promotion
	err := row.Scan(&promotion.ID, &promotion.Price, &promotion.ExpirationDate)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return "not found" response
			http.NotFound(w, r)
		} else {
			// Return an error response
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Convert the promotion to JSON
	jsonData, err := json.Marshal(promotion)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set the response content type and write the JSON data
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
