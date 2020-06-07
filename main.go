package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/dictuantran/Tasks/config"
	"github.com/dictuantran/Tasks/views"
)

func main() {
	values, err := config.ReadConfig("config.json")
	var port *string

	if err != nil {
		port = flag.String("port", "", "IP address")
		flag.Parse()

		//User is expected to give :8080 like input, if they give 8080
		//we'll append the required ':'
		if !strings.HasPrefix(*port, ":") {
			*port = ":" + *port
			log.Println("port is " + *port)
		}

		values.ServerPort = *port
	}

	views.PopulateTemplates()

	// Login logout
	http.HandleFunc("/login/", views.LoginFunc)

	// these handlers are used for restoring tasks
	http.HandleFunc("/", views.RequiresLogin(views.ShowAllTasksFunc))
	http.HandleFunc("/add/", views.RequiresLogin(views.AddTaskFunc))

	http.Handle("/static/", http.FileServer(http.Dir("public")))

	log.Println("running server on ", values.ServerPort)
	log.Fatal(http.ListenAndServe(values.ServerPort, nil))
}
