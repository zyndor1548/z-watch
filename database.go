package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

var Databaseconnection *sql.DB

func ConnectDatabase() (*sql.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./monitoring.db"
	}

	var err error
	Databaseconnection, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	err = Databaseconnection.Ping()
	if err != nil {
		return nil, err
	}
	go CreateDatabase()
	return Databaseconnection, nil
}

func DisconnectDatabase() error {
	if Databaseconnection != nil {
		return Databaseconnection.Close()
	}
	return nil
}

func CreateDatabase() error {

	usersQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		email TEXT DEFAULT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := Databaseconnection.Exec(usersQuery)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	sitesQuery := `
	CREATE TABLE IF NOT EXISTS sites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		url TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	)`

	_, err = Databaseconnection.Exec(sitesQuery)
	if err != nil {
		return fmt.Errorf("failed to create sites table: %w", err)
	}
	checklogsQuery := `
	CREATE TABLE IF NOT EXISTS checklogs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url_id INTEGER NOT NULL,
		latency TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (url_id) REFERENCES sites(id)
	)`

	_, err = Databaseconnection.Exec(checklogsQuery)
	if err != nil {
		return fmt.Errorf("failed to create checklogs table: %w", err)
	}
	return nil
}

func CreateUser(name, password string, done chan bool, token chan string) error {
	query := "INSERT INTO users (name, password) VALUES (?, ?)"
	hashPassword, err := HashPassword(password)
	if err != nil {
		fmt.Printf("error : %v", err)
		return fmt.Errorf("failed to hash password: %w", err)
	}
	result, err := Databaseconnection.Exec(query, name, hashPassword)
	if err != nil {
		fmt.Printf("error : %v", err)
		return fmt.Errorf("failed to insert server: %w", err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		fmt.Printf("error : %v", err)
		return err
	}
	done <- true
	go GenerateToken(lastID, token)
	return nil
}

func ValidateUser(name, password string, done chan bool, token chan string) error {
	query := "SELECT id, name, password FROM users WHERE name = ?"

	var userid int64
	var dbName, dbPassword string

	err := Databaseconnection.QueryRow(query, name).Scan(&userid, &dbName, &dbPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			done <- false
			token <- ""
			return err
		}
		done <- false
		token <- ""
		return err
	}
	if VerifyPassword(password, dbPassword) {
		done <- true
		go GenerateToken(userid, token)
		return nil
	}
	done <- false
	token <- ""
	return nil
}

func AddSiteToDatabase(userID int64, url string, done chan bool) error {
	query := "INSERT INTO sites (user_id, url) VALUES (?, ?)"

	_, err := Databaseconnection.Exec(query, userID, url)
	if err != nil {
		done <- false
		return fmt.Errorf("failed to insert site: %w", err)
	}
	go AddNewMoniter(userID, url)
	done <- true
	return nil
}
func LogCheck(urlid int64, data Response) error {
	query := "INSERT INTO checklogs (url_id, latency, status) VALUES (?, ?, ?)"
	_, err := Databaseconnection.Exec(query, urlid, data.Latency, data.Status)
	if err != nil {
		return fmt.Errorf("failed to insert checklog: %w", err)
	}

	cleanupQuery := `
		DELETE FROM checklogs 
		WHERE url_id = ? 
		AND id NOT IN (
			SELECT id FROM checklogs 
			WHERE url_id = ? 
			ORDER BY created_at DESC 
			LIMIT 30
		)
	`
	_, err = Databaseconnection.Exec(cleanupQuery, urlid, urlid)
	if err != nil {
		fmt.Printf("Warning: failed to cleanup old logs for url_id %d: %v\n", urlid, err)
	}

	return nil
}

type CheckLogEntry struct {
	ID        int64  `json:"id"`
	URLID     int64  `json:"url_id"`
	URL       string `json:"url"`
	Latency   string `json:"latency"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type URLCheckLogs struct {
	URLID int64           `json:"url_id"`
	URL   string          `json:"url"`
	Logs  []CheckLogEntry `json:"logs"`
}

func GetCheckLogs(userid int64, done chan bool, data chan []URLCheckLogs) error {
	query := `SELECT checklogs.id,checklogs.url_id,sites.url,checklogs.latency,checklogs.status,checklogs.created_at FROM checklogs INNER JOIN sites ON checklogs.url_id = sites.id WHERE sites.user_id = ? ORDER BY sites.id, checklogs.created_at DESC`

	rows, err := Databaseconnection.Query(query, userid)
	if err != nil {
		done <- false
		data <- nil
		return fmt.Errorf("failed to query checklogs: %w", err)
	}
	defer rows.Close()

	urlMap := make(map[int64]*URLCheckLogs)
	var urlOrder []int64

	for rows.Next() {
		var log CheckLogEntry
		err := rows.Scan(
			&log.ID,
			&log.URLID,
			&log.URL,
			&log.Latency,
			&log.Status,
			&log.CreatedAt,
		)
		if err != nil {
			done <- false
			data <- nil
			return fmt.Errorf("failed to scan checklog row: %w", err)
		}
		if _, exists := urlMap[log.URLID]; !exists {
			urlMap[log.URLID] = &URLCheckLogs{
				URLID: log.URLID,
				URL:   log.URL,
				Logs:  []CheckLogEntry{},
			}
			urlOrder = append(urlOrder, log.URLID)
		}

		urlMap[log.URLID].Logs = append(urlMap[log.URLID].Logs, log)
	}

	if err = rows.Err(); err != nil {
		done <- false
		data <- nil
		return fmt.Errorf("error iterating checklog rows: %w", err)
	}

	var groupedLogs []URLCheckLogs
	for _, urlID := range urlOrder {
		groupedLogs = append(groupedLogs, *urlMap[urlID])
	}
	done <- true
	data <- groupedLogs
	return nil
}

func LoadWatch(done chan bool) {
	query := "SELECT id,user_id,url FROM sites"
	rows, err := Databaseconnection.Query(query)
	if err != nil {
		fmt.Printf("Error querying sites: %v\n", err)
		done <- false
		return
	}
	defer rows.Close()

	for rows.Next() {
		var monitor Monitor
		var userId int64

		err := rows.Scan(&monitor.Id, &userId, &monitor.Url)
		if err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			continue
		}
		monitor.Interval = (2 * time.Minute)
		monitor.NextRunTime = time.Now()

		ScheduleMutex.Lock()
		ScheduleMonitor = append(ScheduleMonitor, monitor)
		ScheduleMutex.Unlock()
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Error iterating rows: %v\n", err)
	}
	done <- true
}
