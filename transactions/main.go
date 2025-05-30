package main

import (
	"database/sql"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transactions_http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"path", "method", "code"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transactions_http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets, // Default buckets
		},
		[]string{"path", "method"},
	)
)

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate() // Get path template for consistent labeling

		timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(path, r.Method))

		// Use a custom response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK} // Default to 200

		next.ServeHTTP(rw, r) // Serve the request

		statusCode := strconv.Itoa(rw.statusCode)
		httpRequestsTotal.WithLabelValues(path, r.Method, statusCode).Inc()
		timer.ObserveDuration() // Observe duration after request is handled
	})
}

type Transaction struct {
	ID        int       `json:"id"`
	FromID    uuid.UUID `json:"from_id"`
	ToID      uuid.UUID `json:"to_id"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

func initDB() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to database: %q", err)
	}
	log.Println("Successfully connected to database!")
}

func main() {
	initDB() // Initialize database connection and schema
	r := mux.NewRouter()

	// Apply Prometheus middleware to all routes except /metrics
	r.Use(prometheusMiddleware)

	apiRouter := r.PathPrefix("/v1").Subrouter()
	apiRouter.HandleFunc("/transactions", getTransactionsHandler).Methods(http.MethodGet)
	apiRouter.HandleFunc("/transactions", createTransactionHandler).Methods(http.MethodPost)

	// Prometheus metrics endpoint - does not go through the logging/metrics middleware for itself
	// Use a new router or the main router for the /metrics endpoint
	metricsRouter := mux.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler())

	// It's common to serve metrics on a different port or ensure it's not caught by general middleware.
	// For simplicity here, we'll serve it alongside the API.
	// If you want /metrics on the same port and router, add it before r.Use(prometheusMiddleware)
	// or ensure the middleware correctly skips /metrics.
	// The simplest is to have it on the main router before the subrouter for API.
	r.Handle("/metrics", promhttp.Handler())

	port := "8080"
	log.Printf("Server starting on port %s\n", port)
	log.Printf("Metrics available at http://localhost:%s/metrics\n", port)
	log.Printf("API available at http://localhost:%s/v1\n", port)

	// Start server
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}
