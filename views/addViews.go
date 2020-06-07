package views

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dictuantran/Tasks/db"
	"github.com/dictuantran/Tasks/sessions"
)

//AddTaskFunc is used to handle the addition of new task, "/add" URL
func AddTaskFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/", http.StatusBadRequest)
		return
	}

	var filelink string
	r.ParseForm()
	file, handler, err := r.FormFile("uploadfile")
	if err != nil && handler != nil {
		// Case executed when file is uploaded and yet an error occurs
		log.Println(err)
		message = "Error upload file"
		http.Redirect(w, r, "/", http.StatusInternalServerError)
	}

	taskPriority, priorityErr := strconv.Atoi(r.FormValue("priority"))

	if priorityErr != nil {
		log.Print(priorityErr)
		message = "Bad task priority"
	}

	priorityList := []int{1, 2, 3}
	found := false
	for _, priority := range priorityList {
		if taskPriority == priority {
			found = true
		}
	}

	//If someone gives us incorrect priority number, we give the priority
	//to that task as 1 i.e. Low

	if !found {
		taskPriority = 1
	}

	var hidden int
	hideTimeline := r.FormValue("hide")
	if hideTimeline != "" {
		hidden = 1
	} else {
		hidden = 0
	}

	// dueDate := r.FormValue("dueDate")
	category := r.FormValue("category")
	title := template.HTMLEscapeString(r.Form.Get("title"))
	content := template.HTMLEscapeString(r.Form.Get("content"))
	formToken := template.HTMLEscapeString(r.Form.Get("CSRFToken"))

	cookie, _ := r.Cookie("csrftoken")

	if formToken == cookie.Value {
		username := sessions.GetCurrentUserName(r)
		if handler != nil {
			// this will be executed whenever a file is uploaded
			r.ParseMultipartForm(32 << 20) //defined maximum size of file
			defer file.Close()
			htmlFilename := strings.Replace(handler.Filename, " ", "-", -1)
			randomFileName := md5.New()
			io.WriteString(randomFileName, strconv.FormatInt(time.Now().Unix(), 10))
			io.WriteString(randomFileName, htmlFilename)
			token := fmt.Sprintf("%x", randomFileName.Sum(nil))
			f, err := os.OpenFile("./files/"+htmlFilename, os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				log.Println(err)
				return
			}
			defer f.Close()
			io.Copy(f, file)

			if strings.HasSuffix(htmlFilename, ".png") || strings.HasSuffix(htmlFilename, ".jpg") {
				filelink = "<br> <img src='/files/" + htmlFilename + "'/>"
			} else {
				filelink = "<br> <a href=/files/" + htmlFilename + ">" + htmlFilename + "</a>"
			}
			content = content + filelink

			fileTruth := db.AddFile(htmlFilename, token, username)
			if fileTruth != nil {
				message = "Error adding filename in db"
				log.Println("error adding task to db")
			}
		}
		//taskTruth := db.AddTask(title, content, category, taskPriority, username, dueDate)
		taskTruth := db.AddTask(title, content, category, taskPriority, username, hidden)
		if taskTruth != nil {
			message = "Error adding task"
			log.Println("error adding task to db")
			http.Redirect(w, r, "/", http.StatusInternalServerError)
		} else {
			message = "Task added"
			log.Println("added task to db")
			http.Redirect(w, r, "/", http.StatusFound)
		}
	} else {
		log.Println("CSRF mismatch")
		message = "Server Error"
		http.Redirect(w, r, "/", http.StatusInternalServerError)
	}
}
