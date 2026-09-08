package main

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/wxlbd/ruoyi-mall-go/migrations"
	"log"
	"os"
)

func main() {
	dsn := os.Getenv("RUOYI_DATABASE_DSN")
	if dsn == "" {
		log.Fatal("RUOYI_DATABASE_DSN is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = migrations.Apply(db, 0); err != nil {
		log.Fatal(err)
	}
	log.Print("database is at latest migration")
}
