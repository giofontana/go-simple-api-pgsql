package main

import (
    "fmt"
    "log"
    "net/http"
    "encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"os/signal"	
	"time"
	"context"
    "syscall"	
    "github.com/jinzhu/gorm"    
	_ "github.com/jinzhu/gorm/dialects/postgres"    
)

var db *gorm.DB
var err error

type Article struct {
    gorm.Model
    Title   string
    Desc    string
    Content string
}

var (
    articles = []Article{	
        Article{Title: "Hello", Desc: "Article Description", Content: "Article Content"},
        Article{Title: "Hello 2", Desc: "Article Description", Content: "Article Content"},
    }
)


// Existing code from above
func handleRequests() {

	username := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")

	dbUri := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", dbHost, username, dbName, password) //Build connection string
	fmt.Println(dbUri)

	db, err = gorm.Open( "postgres", dbUri)

	if err != nil {
        panic("failed to connect database")
    }

	defer db.Close()

	db.AutoMigrate(&Article{})
  
	for index := range articles {
		db.Create(&articles[index])
	}

    myRouter := mux.NewRouter().StrictSlash(true)
    myRouter.HandleFunc("/", homePage)
    myRouter.HandleFunc("/articles", returnAllArticles)
    // NOTE: Ordering is important here! This has to be defined before
	// the other `/article` endpoint. 
	myRouter.HandleFunc("/article/{id}", returnSingleArticle).Methods("GET")	
    myRouter.HandleFunc("/article", createNewArticle).Methods("POST")
	myRouter.HandleFunc("/article/{id}", deleteArticle).Methods("DELETE")
	myRouter.HandleFunc("/article/{id}", updateArticle).Methods("PUT")

	srv := &http.Server{
		Handler:      myRouter,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}


	log.Println("Starting Server")

	log.Fatal(http.ListenAndServe(":8080", myRouter))
	// Graceful Shutdown
	waitForShutdown(srv)	
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

func main() {

	// Configure Logging
	LOG_FILE_LOCATION := os.Getenv("LOG_FILE_LOCATION")
	if LOG_FILE_LOCATION != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   LOG_FILE_LOCATION,
			MaxSize:    500, // megabytes
			MaxBackups: 3,
			MaxAge:     28,   //days
			Compress:   true, // disabled by default
		})
	}

    handleRequests()
}

func returnSingleArticle(w http.ResponseWriter, r *http.Request){
	params := mux.Vars(r)
	var article Article
	db.First(&article, params["id"])
	json.NewEncoder(w).Encode(&article)
}

func createNewArticle(w http.ResponseWriter, r *http.Request) {
    // get the body of our POST request
    // unmarshal this into a new Article struct
    // append this to our Articles array.    
    reqBody, _ := ioutil.ReadAll(r.Body)
    var article Article 
    json.Unmarshal(reqBody, &article)

	db.Create(&article)

	log.Println("Article ID: ", article.ID)

    json.NewEncoder(w).Encode(article)
}

func deleteArticle(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var article Article
  
	db.First(&article, params["id"])
	db.Delete(&article)

	var articles []Article
	
	db.Find(&articles)
	json.NewEncoder(w).Encode(&articles)

}

func updateArticle(w http.ResponseWriter, r *http.Request){
	params := mux.Vars(r)
	var articleToUpdate Article
 
	db.First(&articleToUpdate, params["id"])

	reqBody, _ := ioutil.ReadAll(r.Body)
	var articleNew Article 
	json.Unmarshal(reqBody, &articleNew)

	articleToUpdate.Title = articleNew.Title
	articleToUpdate.Desc = articleNew.Desc
	articleToUpdate.Content = articleNew.Content

	db.Save(&articleToUpdate)

	json.NewEncoder(w).Encode(&articleToUpdate)

}

func returnAllArticles(w http.ResponseWriter, r *http.Request){
    fmt.Println("Endpoint Hit: returnAllArticles")
	db.Find(&articles)
	json.NewEncoder(w).Encode(&articles)
}

func homePage(w http.ResponseWriter, r *http.Request){
    fmt.Fprintf(w, "Welcome to the HomePage!")
    fmt.Println("Endpoint Hit: homePage")
}