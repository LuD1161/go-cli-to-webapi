package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	uuid "github.com/satori/go.uuid"
)

var db *gorm.DB
var err error

type job struct {
	gorm.Model
	Application string    `json:"application"`
	Status      string    `json:"status"`
	Worker      string    `json:"worker"`
	JobId       uuid.UUID `gorm:"type:uuid" json:"job_id"`
}

// BeforeCreate will set a UUID in the job_id column
func (job *job) BeforeCreate(scope *gorm.Scope) error {
	uuid := uuid.NewV4()
	return scope.SetColumn("job_id", uuid)
}

type Response struct {
	Message string
	Error   string
}

func init() {
	db, err = gorm.Open("postgres", "host=localhost port=5432 user=postgres dbname=webcli sslmode=disable password=postgres")
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate(&job{})
}

func getJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var jobs []job
	if result := db.Find(&jobs); result.Error != nil {
		sendErrorResponse(w, "Error retrieving jobs", err)
	}
	json.NewEncoder(w).Encode(jobs)
}

func getJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	job_id := vars["job_id"]
	var job job
	// basic validation for UUID job_id
	uid, err := uuid.FromString(job_id)
	fmt.Println(uid)
	fmt.Println(err)
	if _, err := uuid.FromString(job_id); err != nil {
		sendErrorResponse(w, "Invalid job id "+job_id, err)
	}
	if result := db.Where("job_id = ?", uid).First(&job); result.Error != nil {
		sendErrorResponse(w, "Error retrieving job with "+job_id, result.Error)
	}
	json.NewEncoder(w).Encode(job)
}

func createJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var job job
	_ = json.NewDecoder(r.Body).Decode(&job)
	if result := db.Save(&job); result.Error != nil {
		sendErrorResponse(w, fmt.Sprintf("Error creating job  %+v", job), err)
	}
	json.NewEncoder(w).Encode(job)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/job", getJobs).Methods("GET")
	r.HandleFunc("/job/{job_id}", getJob).Methods("GET")
	r.HandleFunc("/job", createJob).Methods("POST")

	log.Fatal(http.ListenAndServe(":8000", r))
}

func sendErrorResponse(w http.ResponseWriter, message string, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	if err := json.NewEncoder(w).Encode(Response{Message: message, Error: err.Error()}); err != nil {
		panic(err)
	}
}
