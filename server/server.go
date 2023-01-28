package main

import (
	"database/sql"
	"log"
	"mailinglist/jsonapi"
	"mailinglist/mdb"

	"github.com/alexflint/go-arg"
)

var args struct {
	DbPath string `arg:"env:MAILING_LIST_DB"`
	Bind   string `arg:"env:MAILING_LIST_BIND_PORT"`
}

func main() {
	arg.MustParse(&args)

	// Default values
	if args.DbPath == "" {
		args.DbPath = "list.db"
	}
	if args.Bind == "" {
		args.Bind = ":9091"
	}

	log.Printf("using db path %v and bind address %v\n", args.DbPath, args.Bind)

	db, err := sql.Open("sqlite3", args.DbPath)
	if err != nil {
		log.Fatalf("Error opening sqlite db : %v\n", err)
	}
	defer db.Close()

	mdb.TryCreate(db)

	jsonapi.Serve(db, args.Bind)

}
