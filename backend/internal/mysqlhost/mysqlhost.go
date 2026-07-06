package mysqlhost

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Host struct {
	Hostname      string
	Port          int
	AdminUsername string
	AdminPassword string
}

var identPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func ValidIdentifier(s string) bool {
	return s != "" && len(s) <= 64 && identPattern.MatchString(s)
}

func open(ctx context.Context, h Host) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=10s", h.AdminUsername, h.AdminPassword, h.Hostname, h.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(10 * time.Second)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func Ping(ctx context.Context, h Host) error {
	db, err := open(ctx, h)
	if err != nil {
		return err
	}
	return db.Close()
}

func Provision(ctx context.Context, h Host, dbName, username, password string) error {
	if !ValidIdentifier(dbName) || !ValidIdentifier(username) {
		return fmt.Errorf("invalid database or username identifier")
	}

	db, err := open(ctx, h)
	if err != nil {
		return fmt.Errorf("connect to database host: %w", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+dbName+"`"); err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE USER IF NOT EXISTS '"+username+"'@'%' IDENTIFIED BY '"+password+"'"); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	if _, err := db.ExecContext(ctx, "GRANT ALL PRIVILEGES ON `"+dbName+"`.* TO '"+username+"'@'%'"); err != nil {
		return fmt.Errorf("grant privileges: %w", err)
	}
	if _, err := db.ExecContext(ctx, "FLUSH PRIVILEGES"); err != nil {
		return fmt.Errorf("flush privileges: %w", err)
	}
	return nil
}

func Deprovision(ctx context.Context, h Host, dbName, username string) error {
	if !ValidIdentifier(dbName) || !ValidIdentifier(username) {
		return fmt.Errorf("invalid database or username identifier")
	}

	db, err := open(ctx, h)
	if err != nil {
		return fmt.Errorf("connect to database host: %w", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+dbName+"`"); err != nil {
		return fmt.Errorf("drop database: %w", err)
	}
	if _, err := db.ExecContext(ctx, "DROP USER IF EXISTS '"+username+"'@'%'"); err != nil {
		return fmt.Errorf("drop user: %w", err)
	}
	return nil
}
