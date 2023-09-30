package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/lib/pq"
)

type Record struct {
	ID      int    `json:"id"`
	AlertID int    `json:"alert_id"`
	Label   string `json:"label"`
	Value   string `json:"value"`
}

var db *sql.DB

func init() {
	// Initialize the database connection
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432" // Default PostgreSQL port
	}

	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")

	connStr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	http.HandleFunc("/api/create", handleCreate)
	http.HandleFunc("/api/records", handleRecords)
	http.HandleFunc("/api/update", handleUpdate)
	http.HandleFunc("/api/delete", handleDelete)

	log.Printf("Starting server on port :8080\n")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Fetch records from the database
		records, err := fetchRecords()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Encode records as newline-separated JSON and send the response
		w.Header().Set("Content-Type", "application/json")
		for i, record := range records {
			recordJSON, err := json.Marshal(record)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if i > 0 {
				w.Write([]byte("\n")) // Add newline separator between records
			}
			w.Write(recordJSON)
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func fetchRecords() ([]Record, error) {
	// Fetch records from the table
	rows, err := db.Query("SELECT id, alertid, label, value FROM alertlabel")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record

	for rows.Next() {
		var record Record
		if err := rows.Scan(&record.ID, &record.AlertID, &record.Label, &record.Value); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		// Extract the number of records to delete from the user
		numRecordsParam := r.URL.Query().Get("num_records")
		if numRecordsParam == "" {
			http.Error(w, "Missing 'num_records' parameter", http.StatusBadRequest)
			return
		}

		// Convert the numRecords parameter to an integer
		numRecords, err := strconv.Atoi(numRecordsParam)
		if err != nil {
			http.Error(w, "Invalid 'num_records' parameter", http.StatusBadRequest)
			return
		}

		// Delete the specified number of records in a loop
		for i := 0; i < numRecords; i++ {

			err := deleteRecord(i)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Respond with a success message
		w.WriteHeader(http.StatusNoContent)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func deleteRecord(id int) error {
	// Delete the record with the specified ID from the table
	// /api/delete?id=123 Delete one records from API
	// curl -X DELETE "http://localhost:8080/api/records/delete?num_records=300"
	// _, err := db.Exec("DELETE FROM alertlabel WHERE id = $1", id)
	_, err := db.Exec("DELETE FROM alertlabel WHERE id = (SELECT MIN(id) FROM alertlabel)")
	return err
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Parse the request body to get the update data
		var data Record
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = updateRecord(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Updated record with ID %d and AlertID %d", data.ID, data.AlertID)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// curl -X POST -H "Content-Type: application/json" -d '{"ID": 10301, "AlertID": 1210, "Label": "Updated Label", "Value": "Updated Value"}' http://localhost:8080/api/update

func updateRecord(data Record) error {
	// Update the record in the database based on the provided data
	_, err := db.Exec("UPDATE alertlabel SET label = $1, value = $2 WHERE id = $3 AND alertid = $4",
		data.Label, data.Value, data.ID, data.AlertID)
	return err
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Parse the request body to get the create data
		var record Record
		err := json.NewDecoder(r.Body).Decode(&record)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = createRecord(record)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "Created record with ID %d and AlertID %d", record.ID, record.AlertID)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// curl -X POST -H "Content-Type: application/json" -d '{"ID": 1, "AlertID": 1, "Label": "New Label", "Value": "New Value"}' http://localhost:8080/api/create

func createRecord(record Record) error {
	// Insert a new record into the database
	_, err := db.Exec("INSERT INTO alertlabel (id, alertid, label, value) VALUES ($1, $2, $3, $4)",
		record.ID, record.AlertID, record.Label, record.Value)
	return err
}
