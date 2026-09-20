package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/term"
)

//go:embed config.json
var configData []byte

type Config struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	DBName  string `json:"dbname"`
	SSLMode string `json:"sslmode"`
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	userLogin, userPassword, err := promptCredentials()
	if err != nil {
		log.Fatalf("creds: %v", err)
	}

	conn, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig("")
	if err != nil {
		log.Fatalf("pase pool config: %v", err)
	}

	poolConfig.ConnConfig.Host = cfg.Host
	poolConfig.ConnConfig.Port = uint16(cfg.Port)
	poolConfig.ConnConfig.User = userLogin
	poolConfig.ConnConfig.Password = userPassword
	poolConfig.ConnConfig.Database = cfg.DBName

	pool, err := pgxpool.NewWithConfig(conn, poolConfig)
	if err != nil {
		log.Fatalf("create pool: %v", err)
	}

	defer pool.Close()

	if err := pool.Ping(conn); err != nil {
		log.Fatalf("ping: %v", err)
	}
	log.Printf("Successful connection to %s:%d/%s as %s", cfg.Host, cfg.Port, cfg.DBName, userLogin)

	var version string
	if err := pool.QueryRow(conn, "SELECT VERSION();").Scan(&version); err != nil {
		log.Fatalf("query: %v", err)
	}
	log.Printf("Postgres Version %s", version)
}

func loadConfig() (*Config, error) {

	var cfg Config
	if err := json.Unmarshal(configData, &cfg); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}
	if cfg.Host == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("host and dbname is nessesary")
	}
	return &cfg, nil
}

func promptCredentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Login:")
	userLogin, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	userLogin = strings.TrimSpace(userLogin)

	fmt.Print("Enter Password:")
	bytePwd, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", "", err
	}

	if userLogin == "" || len(bytePwd) == 0 {
		return "", "", fmt.Errorf("Login or Password is empty")
	}
	return userLogin, string(bytePwd), nil
}
