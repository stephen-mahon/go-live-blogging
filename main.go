package main

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	_ "github.com/mattn/go-sqlite3"
	pusher "github.com/pusher/pusher-http-go"
)

var client = pusher.Client{
	AppID:   "1267571",
	Key:     "bedd75861b120f0ff178",
	Secret:  "9969d5cc6e9c10c64a97",
	Cluster: "us2",
	Secure:  true,
}

// Post type
type Post struct {
	ID      int64  `json:"id"`
	Content string `json:"content"`
}

// PostCollection type
type PostCollection struct {
	Posts []Post `json:"items"`
}

func initialiseDatabase(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		panic(err)
	}

	if db == nil {
		panic("db nil")
	}

	return db
}

func migrateDatabase(db *sql.DB) {
	sql := `
		CREATE TABLE IF NOT EXISTS posts(
				id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
				content TEXT
		);
`
	_, err := db.Exec(sql)
	if err != nil {
		panic(err)
	}
}

func getPosts(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows, err := db.Query("SELECT * FROM posts ORDER BY id DESC")
		if err != nil {
			panic(err)
		}

		defer rows.Close()

		result := PostCollection{}

		for rows.Next() {
			post := Post{}
			err2 := rows.Scan(&post.ID, &post.Content)
			if err2 != nil {
				panic(err2)
			}

			result.Posts = append(result.Posts, post)
		}

		return c.JSON(http.StatusOK, result)
	}
}

func savePost(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		postContent := c.FormValue("content")
		stmt, err := db.Prepare("INSERT INTO posts (content) VALUES(?)")
		if err != nil {
			panic(err)
		}

		defer stmt.Close()

		result, err := stmt.Exec(postContent)
		if err != nil {
			panic(err)
		}

		insertedID, err := result.LastInsertId()
		if err != nil {
			panic(err)
		}

		post := Post{
			ID:      insertedID,
			Content: postContent,
		}

		client.Trigger("live-blog-stream", "new-post", post)
		return c.JSON(http.StatusOK, post)
	}
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	db := initialiseDatabase("./database/storage.db")
	migrateDatabase(db)

	e.File("/", "public/index.html")
	e.File("/admin", "public/admin.html")
	e.GET("/posts", getPosts(db))
	e.POST("/posts", savePost(db))

	e.Logger.Fatal(e.Start(":9000"))
}
