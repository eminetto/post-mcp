package main

import (
	"encoding/json"
	"github.com/lib/pq"
	"net/http"
)

// Custom responseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func getTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT * FROM transactions ORDER BY id ASC")
	if err != nil {
		http.Error(w, "Failed to retrieve transactions: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	trs := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.FromID, &t.ToID, &t.Amount, &t.CreatedAt); err != nil {
			http.Error(w, "Failed to scan transaction data: "+err.Error(), http.StatusInternalServerError)
			return
		}
		trs = append(trs, t)
	}
	if err = rows.Err(); err != nil {
		http.Error(w, "Error during rows iteration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trs)
}

func createTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var t Transaction
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Basic validation
	if t.FromID.String() == "" || t.ToID.String() == "" || t.Amount < 0 {
		http.Error(w, "Missing or invalid transaction data", http.StatusBadRequest)
		return
	}

	sqlStatement := `
        INSERT INTO transactions (from_id, to_id, amount)
        VALUES ($1, $2, $3)
        RETURNING id, created_at`
	err := db.QueryRow(sqlStatement, t.FromID, t.ToID, t.Amount).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		// Check for unique constraint violation (e.g., for ISBN)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // 23505 is unique_violation
			http.Error(w, "Failed to create transaction: "+err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}
