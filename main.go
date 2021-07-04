package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

var db *gorm.DB
var err error
var output chan Job

const (
	APP_PORT = "8000"
)

type Job struct {
	gorm.Model `json:"-"`
	CMDString  string    `json:"cmd_string"`
	Status     int       `json:"status"`
	Worker     string    `json:"worker"`
	JobId      uuid.UUID `gorm:"type:uuid" json:"job_id"`
	Output     string    `json:"output"` // save job_output_file
}

// BeforeCreate will set a UUID in the job_id column
func (job *Job) BeforeCreate(scope *gorm.Scope) error {
	uuid := uuid.NewV4()
	return scope.SetColumn("job_id", uuid)
}

type Response struct {
	Message string
	Error   string
}

func init() {
	log.Info("Setting up new database!!!")
	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbTable := os.Getenv("DB_TABLE")
	dbPort := os.Getenv("DB_PORT")
	sslMode := os.Getenv("SSL_MODE")

	connectString := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s", dbHost, dbPort, dbUsername, dbTable, dbPassword, sslMode)
	db, err = gorm.Open("postgres", connectString)
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate(&Job{})
	log.SetFormatter(&log.JSONFormatter{})
	log.Info("Database Initialized")
}

func StatusUpdater(output chan Job) {
	for {
		select {
		case job := <-output:
			log.Info("Job Completed : ", job.JobId)
			db.Model(&job).Updates(&job)
		}
	}
}

func getJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var jobs []Job
	if result := db.Find(&jobs); result.Error != nil {
		sendErrorResponse(w, "Error retrieving jobs", err)
		return
	}
	json.NewEncoder(w).Encode(jobs)
}

func getJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	job_id := vars["job_id"]
	var job Job

	// basic validation for UUID job_id
	if _, err := uuid.FromString(job_id); err != nil {
		sendErrorResponse(w, "Invalid job id "+job_id, err)
		return
	}

	if result := db.Where("job_id = ?", job_id).First(&job); result.Error != nil {
		sendErrorResponse(w, "Error retrieving job with "+job_id, result.Error)
		return
	}

	json.NewEncoder(w).Encode(job)
}

func createJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var job Job
	_ = json.NewDecoder(r.Body).Decode(&job)

	if result := db.Save(&job); result.Error != nil {
		sendErrorResponse(w, fmt.Sprintf("Error creating job  %+v", job), err)
		return
	}

	// start a worker for this
	go Worker(job, output)

	if err != nil {
		sendErrorResponse(w, fmt.Sprintf("Error creating job  %+v", job), err)
	}

	json.NewEncoder(w).Encode(job.JobId)
}

// LoggingMiddleware - adds middleware around endpoints
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(
			log.Fields{
				"Method": r.Method,
				"Path":   r.URL.Path,
			}).Info("Handled request")
		next.ServeHTTP(w, r)
	})
}

func health_check(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]int{"ok": 1})
}

func main() {
	r := mux.NewRouter()
	r.Use(LoggingMiddleware)
	r.HandleFunc("/health_check", health_check).Methods("GET")
	r.HandleFunc("/job", getJobs).Methods("GET")
	r.HandleFunc("/job/{job_id}", getJob).Methods("GET")
	r.HandleFunc("/job", createJob).Methods("POST")

	output = make(chan Job, 100)
	// Create a status updater function
	go StatusUpdater(output)
	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + APP_PORT,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	// Start Server
	go func() {
		log.Println("Starting Server")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Graceful Shutdown
	waitForShutdown(srv)
}

func sendErrorResponse(w http.ResponseWriter, message string, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	if err := json.NewEncoder(w).Encode(Response{Message: message, Error: err.Error()}); err != nil {
		panic(err)
	}
}

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}
