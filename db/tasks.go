package db

import (
	"database/sql"
	"html/template"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/dictuantran/Tasks/types"
	_ "github.com/go-sql-driver/mysql"
	md "github.com/shurcooL/github_flavored_markdown"
)

var database Database
var taskStatus map[string]int
var err error

// Database encapsulates database
type Database struct {
	db *sql.DB
}

// Begins a transaction
func (db Database) begin() (tx *sql.Tx) {
	tx, err := db.db.Begin()
	if err != nil {
		log.Println(err)
		return nil
	}
	return tx
}

func (db Database) prepare(q string) (stmt *sql.Stmt) {
	stmt, err := db.db.Prepare(q)
	if err != nil {
		log.Println(err)
		return nil
	}
	return stmt
}

func (db Database) query(q string, args ...interface{}) (rows *sql.Rows) {
	rows, err := db.db.Query(q, args...)
	if err != nil {
		log.Println(err)
		return nil
	}
	return rows
}

func init() {
	database.db, err = sql.Open("mysql", "root:1234567@/dbtasks")
	taskStatus = map[string]int{"COMPLETE": 1, "PENDING": 2, "DELETED": 3}
	if err != nil {
		log.Fatal(err)
	}
}

// Close function closes this database connection
func Close() {
	database.db.Close()
}

//GetTasks retrieves all the tasks depending on the
//status pending or trashed or completed
func GetTasks(username, status, category string) (types.Context, error) {
	log.Println("getting tasks for ", status)
	var tasks []types.Task
	var task types.Task
	var TaskCreated time.Time
	var context types.Context
	var getTaskSQL string
	var rows *sql.Rows

	comments, err := GetComments(username)

	if err != nil {
		return context, err
	}

	//basicSQL := "select t.id, title, content, created_date, priority, case when c.name is null then 'NA' else c.name end from task t, status s, user u left outer join  category c on c.id=t.cat_id where u.username=? and s.id=t.task_status_id and u.id=t.user_id "

	basicSQL := "select t.id, title, content, created_date, priority, "
	basicSQL = basicSQL + "case when c.name is null then 'NA' else c.name end "
	basicSQL = basicSQL + "from user u, status s, task t left outer join category c on c.id = t.cat_id "
	basicSQL = basicSQL + "where u.username=? and s.id=t.task_status_id and u.id=t.user_id"

	if category == "" {
		switch status {
		case "pending":
			getTaskSQL = basicSQL + " and s.status='PENDING' and t.hide!=1"
		case "deleted":
			getTaskSQL = basicSQL + " and s.status='DELETED' and t.hide!=1"
		case "completed":
			getTaskSQL = basicSQL + " and s.status='COMPLETE' and t.hide!=1"
		}

		getTaskSQL += " order by t.created_date asc"

		rows = database.query(getTaskSQL, username)
	} else {
		status = category
		//This is a special case for showing tasks with null categories, we do a union query
		if category == "UNCATEGORIZED" {
			getTaskSQL = "select t.id, title, content, created_date, priority, 'UNCATEGORIZED' from task t, status s, user u where u.username=? and s.id=t.task_status_id and u.id=t.user_id and t.cat_id=0  and  s.status='PENDING'  order by priority desc, created_date asc, finish_date asc"
			rows, err = database.db.Query(getTaskSQL, username)
		} else {
			getTaskSQL = basicSQL + " and name = ?  and  s.status='PENDING'  order by priority desc, created_date asc, finish_date asc"
			rows, err = database.db.Query(getTaskSQL, username, category)
		}

		if err != nil {
			log.Println("tasks.go: something went wrong while getting query fetch tasks by category")
		}
	}

	defer rows.Close()
	for rows.Next() {
		task = types.Task{}

		err = rows.Scan(&task.Id, &task.Title, &task.Content, &TaskCreated, &task.Priority, &task.Category)

		taskCompleted := 0
		totalTasks := 0

		if strings.HasPrefix(task.Content, "- [") {
			for _, value := range strings.Split(task.Content, "\n") {
				if strings.HasPrefix(value, "- [x]") {
					taskCompleted += 1
				}
				totalTasks += 1
			}
			task.CompletedMsg = strconv.Itoa(taskCompleted) + " complete out of " + strconv.Itoa(totalTasks)
		}

		task.ContentHTML = template.HTML(md.Markdown([]byte(task.Content)))
		// TaskContent = strings.Replace(TaskContent, "\n", "<br>", -1)
		if err != nil {
			log.Println(err)
		}

		if comments[task.Id] != nil {
			task.Comments = comments[task.Id]
		}

		TaskCreated = TaskCreated.Local()
		// if task.Priority != "1" { // if priority is not 1 then calculate, else why bother?
		// CurrentTime := time.Now().Local()
		// diff := CurrentTime.Sub(TaskCreated).Hours()
		// if diff > 168 {
		// 	task.IsOverdue = true // If one week then overdue by default
		// }
		// }
		task.Created = TaskCreated.Format("Jan 2 2006")

		tasks = append(tasks, task)
	}
	context = types.Context{Tasks: tasks, Navigation: status}
	return context, nil
}

//AddTask is used to add the task in the database
//TODO: add dueDate feature later
func AddTask(title, content, category string, taskPriority int, username string, hidden int) error {
	log.Println("AddTask: started function")
	var err error
	userID, err := GetUserID(username)
	if err != nil && (title != "" || content != "") {
		return err
	}

	if category == "" {
		sql := `insert into task(title, content, priority, task_status_id, created_date, last_modified_at, user_id,hide) 
		values(?, ?, ?, ?, now(), now(), ?, ?)`

		err = taskQuery(sql, title, content, taskPriority, taskStatus["PENDING"], userID, hidden)
	} else {
		categoryID := GetCategoryByName(username, category)
		sql := `insert into task(title, content, priority, created_date, last_modified_at, cat_id, task_status_id, user_id,hide)
		values(?,?,?,now(), now(), ?,?,?,?)`

		err = taskQuery(sql, title, content, taskPriority, categoryID, taskStatus["PENDING"], userID, hidden)
	}
	return err
}

//GetComments is used to get comments, all of them.
//We do not want 100 different pages to show tasks, we want to use as few pages as possible
//so we are going to populate everything on the damn home pages
func GetComments(username string) (map[int][]types.Comment, error) {
	commentMap := make(map[int][]types.Comment)

	var taskID int
	var comment types.Comment
	var created time.Time

	userID, err := GetUserID(username)
	if err != nil {
		return commentMap, err
	}
	stmt := "select c.id, c.taskID, c.content, c.created, u.username from comments c, task t, user u where t.id=c.taskID and c.user_id=t.user_id and t.user_id=u.id and u.id=?"
	rows := database.query(stmt, userID)

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&comment.ID, &taskID, &comment.Content, &created, &comment.Username)
		comment.Content = string(md.Markdown([]byte(comment.Content)))
		if err != nil {
			return commentMap, err
		}
		// comment.Content = string(md.Markdown([]byte(comment.Content))) ## have to fix the <p> issue markdown support
		created = created.Local()
		comment.Created = created.Format("Jan 2 2006 15:04:05")
		commentMap[taskID] = append(commentMap[taskID], comment)
	}
	return commentMap, nil
}

//CompleteTask  is used to mark tasks as complete
func CompleteTask(username string, id int) error {
	err := taskQuery("update task set task_status_id=?, finish_date=datetime(),last_modified_at=datetime() where id=? and user_id=(select id from user where username=?) ", taskStatus["COMPLETE"], id, username)
	return err
}

//taskQuery encapsulates running multiple queries which don't do much things
func taskQuery(sql string, args ...interface{}) error {
	log.Print("inside task query")
	SQL := database.prepare(sql)
	tx := database.begin()
	_, err = tx.Stmt(SQL).Exec(args...)
	if err != nil {
		log.Println("taskQuery: ", err)
		tx.Rollback()
	} else {
		err = tx.Commit()
		if err != nil {
			log.Println(err)
			return err
		}
		log.Println("Commit successful")
	}
	return err
}

//GetCategoryIDByName will return the category ID for the category, used in the edit task
//function where we need to be able to update the categoryID of the task
func GetCategoryIDByName(username string, category string) int {
	var categoryID int
	getTaskSQL := "select c.id from category c , user u where u.id = c.user_id and name=? and u.username=?"

	rows := database.query(getTaskSQL, category, username)
	defer rows.Close()
	if rows.Next() {
		err := rows.Scan(&categoryID)
		if err != nil {
			log.Println(err)
			//send email to respective people
		}
	}

	return categoryID
}

//UpdateTask is used to update the tasks in the database
func UpdateTask(id int, title, content, category string, priority int, username string, hidden int) error {
	categoryID := GetCategoryIDByName(username, category)
	userID, err := GetUserID(username)
	if err != nil {
		return err
	}
	err = taskQuery("update task set title=?, content=?, cat_id=?, priority = ? where id=? and user_id=?", title, content, categoryID, priority, id, userID)
	return err
}

//SearchTask is used to return the search results depending on the query
func SearchTask(username, query string) (types.Context, error) {
	var tasks []types.Task
	var task types.Task
	var TaskCreated time.Time
	var context types.Context

	comments, err := GetComments(username)
	if err != nil {
		log.Println("SearchTask: something went wrong in finding comments")
	}

	userID, err := GetUserID(username)
	if err != nil {
		return context, err
	}

	stmt := "select t.id, title, content, created_date, priority, c.name from task t, category c where t.user_id=? and c.id = t.cat_id and (title like '%" + query + "%' or content like '%" + query + "%') order by created_date desc"

	rows := database.query(stmt, userID, query, query)
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&task.Id, &task.Title, &task.Content, &TaskCreated, &task.Priority, &task.Category)
		if err != nil {
			log.Println(err)
		}

		if comments[task.Id] != nil {
			task.Comments = comments[task.Id]
		}

		task.Title = strings.Replace(task.Title, query, "<span class='highlight'>"+query+"</span>", -1)
		task.Content = strings.Replace(task.Content, query, "<span class='highlight'>"+query+"</span>", -1)
		task.Content = string(md.Markdown([]byte(task.Content)))

		TaskCreated = TaskCreated.Local()
		CurrentTime := time.Now().Local()
		week := TaskCreated.AddDate(0, 0, 7)

		if (week.String() < CurrentTime.String()) && (task.Priority != "1") {
			task.IsOverdue = true // If one week then overdue by default
		}
		task.Created = TaskCreated.Format("Jan 2 2006")

		tasks = append(tasks, task)
	}
	context = types.Context{Tasks: tasks, Search: query, Navigation: "search"}
	return context, nil
}
