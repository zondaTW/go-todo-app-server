package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
	_ "github.com/lib/pq"
)

const (
	host     = "localdocker"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "todoDB"
)

type Todo struct {
	ID          int    `json:"id" form:"id"`
	Title       string `json:"title" form:"title"`
	Description string `json:"description" form:"description"`
}

type Todos []Todo

func queryTodoTable(db *sql.DB) Todos {
	sqlStatement := "SELECT id, title, description from todo;"
	rows, err := db.Query(sqlStatement)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	todos := make(Todos, 0)

	for rows.Next() {
		var todo Todo
		switch err := rows.Scan(&todo.ID, &todo.Title, &todo.Description); err {
		case nil:
			todos = append(todos, todo)
		default:
			if err != nil {
				panic(err)
			}
		}
	}

	return todos
}

func addTodo(db *sql.DB, todo Todo) Todo {
	var newTodo Todo
	sqlStatement := "INSERT INTO todo (title, description) VALUES ($1, $2) RETURNING id, title, description;"
	err := db.QueryRow(sqlStatement, todo.Title, todo.Description).Scan(&newTodo.ID, &newTodo.Title, &newTodo.Description)
	if err != nil {
		panic(err)
	}
	return newTodo
}

func updateTodo(db *sql.DB, todo Todo) {
	sqlStatement := "UPDATE todo SET title = $2, description = $3 WHERE id = $1;"
	_, err := db.Exec(sqlStatement, todo.ID, todo.Title, todo.Description)
	if err != nil {
		panic(err)
	}
}

func getTodo(db *sql.DB, id string) Todo {
	var todo Todo
	sqlStatement := "SELECT id, title, description from todo where id = $1;"
	err := db.QueryRow(sqlStatement, id).Scan(&todo.ID, &todo.Title, &todo.Description)
	if err != nil {
		panic(err)
	}

	return todo
}

func deleteTodo(db *sql.DB, id string) {
	sqlStatement := "DELETE FROM todo WHERE id = $1;"
	_, err := db.Exec(sqlStatement, id)
	if err != nil {
		panic(err)
	}
}

var db *sql.DB

func main() {
	cfg, err := ini.Load("env.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	var (
		host     = cfg.Section("").Key("host").String()
		port     = cfg.Section("").Key("port").Value()
		user     = cfg.Section("").Key("user").String()
		password = cfg.Section("").Key("password").String()
		dbname   = cfg.Section("").Key("dbname").String()
	)

	connectionString := fmt.Sprintf("host=%s port=%s "+
		"user=%s password=%s "+
		"dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err = sql.Open("postgres", connectionString)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	router := gin.Default()
	router.Use(CORSMiddleware)
	api := router.Group("/api")
	{
		api.GET("/todos", todosGET)
		api.POST("/todos", todosPOST)
		api.PUT("/todos", todoPUT)
		api.GET("/todos/:id", todoGET)
		api.DELETE("/todos/:id", todoDELETE)
	}
	router.Run(":8081")
}

func CORSMiddleware(context *gin.Context) {
	context.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	context.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	context.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	context.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, PUT, DELETE")

	if context.Request.Method == "OPTIONS" {
		context.AbortWithStatus(204)
		return
	}

	context.Next()
}

func todosGET(context *gin.Context) {
	todos := queryTodoTable(db)
	context.JSON(http.StatusOK, todos)
}

func todosPOST(context *gin.Context) {
	var todo Todo
	var newTodo Todo
	if err := context.ShouldBindJSON(&todo); err == nil {
		newTodo = addTodo(db, todo)
		context.JSON(http.StatusOK, newTodo)
	} else {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func todoPUT(context *gin.Context) {
	var todo Todo
	if err := context.ShouldBindJSON(&todo); err == nil {
		updateTodo(db, todo)
		context.JSON(http.StatusOK, gin.H{})
	} else {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func todoGET(context *gin.Context) {
	id := context.Param("id")
	todo := getTodo(db, id)
	context.JSON(http.StatusOK, todo)
}

func todoDELETE(context *gin.Context) {
	id := context.Param("id")
	deleteTodo(db, id)
	context.JSON(http.StatusNoContent, gin.H{})
}
