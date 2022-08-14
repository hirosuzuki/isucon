package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var db *sqlx.DB

func getEnv(key string, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}

func connectDB() (*sqlx.DB, error) {
	config := mysql.NewConfig()
	config.Net = "tcp"
	config.Addr = getEnv("ISUCON_DB_HOST", "127.0.0.1") + ":" + getEnv("ISUCON_DB_PORT", "3306")
	config.User = getEnv("ISUCON_DB_USER", "isuconapp")
	config.Passwd = getEnv("ISUCON_DB_PASSWORD", "isunageruna")
	config.DBName = getEnv("ISUCON_DB_NAME", "isucon")
	config.ParseTime = true
	dsn := config.FormatDSN()
	return sqlx.Open("mysql", dsn)
}

type Article struct {
	ID        int64     `db:"id"`
	Title     string    `db:"title"`
	Body      string    `db:"body"`
	CreatedAt time.Time `db:"created_at"`
}

type Comment struct {
	ID        int64     `db:"id"`
	Article   int64     `db:"article"`
	Name      string    `db:"name"`
	Body      string    `db:"body"`
	CreatedAt time.Time `db:"created_at"`
}

func main() {
	var err error

	e := echo.New()
	e.Debug = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	db, err = connectDB()
	if err != nil {
		e.Logger.Fatalf("failed to connect db: %v", err)
		return
	}
	db.SetMaxOpenConns(10)
	defer db.Close()

	e.Static("/", "public")
	e.GET("/", IndexHandler)
	e.GET("/article/:articleid", articleHandler)
	e.GET("/post", articlePostInputHandler)
	e.POST("/post", articlePostSubmitHandler)
	e.POST("/comment/:articleid", commentPostSubmitHandler)

	e.HTTPErrorHandler = errorResponseHandler

	port := getEnv("SERVER_APP_PORT", "3000")
	e.Logger.Infof("starting isucon server on : %s ...", port)
	serverPort := fmt.Sprintf(":%s", port)
	e.Logger.Fatal(e.Start(serverPort))
}

type FailureResult struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

func errorResponseHandler(err error, c echo.Context) {
	var he *echo.HTTPError
	if errors.As(err, &he) {
		c.JSON(he.Code, FailureResult{
			Status: false,
		})
		return
	}
	c.Logger().Errorf("error at %s: %s", c.Path(), err.Error())
	c.JSON(http.StatusInternalServerError, FailureResult{
		Status: false,
	})
}

func getRecentCommentedArticles() ([]Article, error) {
	article := []Article{}
	query := "SELECT a.id, a.title FROM comment c INNER JOIN article a ON c.article = a.id GROUP BY a.id ORDER BY MAX(c.created_at) DESC LIMIT 10"
	err := db.Select(&article, query)
	return article, err
}

// GET /
func IndexHandler(c echo.Context) error {
	t, err := template.New("base.html").Funcs(template.FuncMap{
		"splitlines": func(s string) []string {
			return strings.Split(s, "\n")
		},
		"date": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
	}).ParseFiles("templates/index.html", "templates/base.html")
	if err != nil {
		log.Fatalf("IndexHandler template error: %v", err)
	}

	recentCommentedArticles, err := getRecentCommentedArticles()
	if err != nil {
		return fmt.Errorf("IndexHandler getRecentCommentedArticles error: %w", err)
	}

	query := "SELECT id,title,body,created_at FROM article ORDER BY id DESC LIMIT 10"
	articles := []Article{}
	err = db.Select(&articles, query)
	if err != nil {
		return fmt.Errorf("IndexHandler get articles error: %w", err)
	}

	writer := new(strings.Builder)
	err = t.Execute(writer, map[string]interface{}{
		"RecentCommentedArticles": recentCommentedArticles,
		"Articles":                articles,
	})
	if err != nil {
		return fmt.Errorf("IndexHandler template exec error: %w", err)
	}

	return c.HTML(http.StatusOK, writer.String())
}

// GET /article/:articleid
func articleHandler(c echo.Context) error {
	t, err := template.New("base.html").Funcs(template.FuncMap{
		"splitlines": func(s string) []string {
			return strings.Split(s, "\n")
		},
		"date": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
	}).ParseFiles("templates/article.html", "templates/base.html")
	if err != nil {
		log.Fatalf("articleHandler template error: %v", err)
	}

	recentCommentedArticles, err := getRecentCommentedArticles()
	if err != nil {
		return fmt.Errorf("articleHandler getRecentCommentedArticles error: %w", err)
	}

	articleID, err := strconv.Atoi(c.Param("articleid"))
	if err != nil {
		return fmt.Errorf("articleHandler Atoi error: %w", err)
	}

	query := "SELECT id,title,body,created_at FROM article WHERE id=?"
	article := Article{}
	err = db.Get(&article, query, articleID)
	if err != nil {
		return fmt.Errorf("articleHandler get Article error: %w", err)
	}

	query = "SELECT name,body,created_at FROM comment WHERE article=? ORDER BY id"
	comments := []Comment{}
	err = db.Select(&comments, query, articleID)
	if err != nil {
		return fmt.Errorf("articleHandler get Comments error: %w", err)
	}

	writer := new(strings.Builder)
	err = t.Execute(writer, map[string]interface{}{
		"RecentCommentedArticles": recentCommentedArticles,
		"Article":                 article,
		"Comments":                comments,
	})
	if err != nil {
		return fmt.Errorf("articleHandler template Exec error: %w", err)
	}

	return c.HTML(http.StatusOK, writer.String())
}

// GET /post
func articlePostInputHandler(c echo.Context) error {
	t, err := template.ParseFiles("templates/post.html", "templates/base.html")
	if err != nil {
		log.Fatalf("articlePostInputHandler template error: %v", err)
	}

	recentCommentedArticles, err := getRecentCommentedArticles()
	if err != nil {
		return fmt.Errorf("articlePostInputHandler getRecentCommentedArticles error: %w", err)
	}

	writer := new(strings.Builder)
	err = t.Execute(writer, map[string]interface{}{
		"RecentCommentedArticles": recentCommentedArticles,
	})
	if err != nil {
		return fmt.Errorf("articlePostInputHandler template Exec error: %w", err)
	}

	return c.HTML(http.StatusOK, writer.String())
}

type PostArticle struct {
	Title string `form:"title"`
	Body  string `form:"body"`
}

// POST /post
func articlePostSubmitHandler(c echo.Context) error {
	postArticle := new(PostArticle)
	err := c.Bind(postArticle)
	if err != nil {
		return fmt.Errorf("articlePostSubmitHandler bind postArticle error: %w", err)
	}

	query := "INSERT INTO article SET title = ?, body = ?"
	_, err = db.Exec(query, postArticle.Title, postArticle.Body)
	if err != nil {
		return fmt.Errorf("articlePostSubmitHandler insert article error: %w", err)
	}

	return c.Redirect(http.StatusMovedPermanently, "/")
}

type PostComment struct {
	Name string `form:"name"`
	Body string `form:"body"`
}

// POST /comment/:articleid
func commentPostSubmitHandler(c echo.Context) error {
	articleID, err := strconv.Atoi(c.Param("articleid"))
	if err != nil {
		return fmt.Errorf("commentPostSubmitHandler Atoi error: %w", err)
	}

	postComment := new(PostComment)
	err = c.Bind(postComment)
	if err != nil {
		return fmt.Errorf("commentPostSubmitHandler bind postComment error: %w", err)
	}

	query := "INSERT INTO comment SET article = ?, name =?, body = ?"
	_, err = db.Exec(query, articleID, postComment.Name, postComment.Body)
	if err != nil {
		return fmt.Errorf("commentPostSubmitHandler insert comment error: %w", err)
	}

	return c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/article/%d", articleID))
}
