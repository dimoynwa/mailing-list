package main

import (
	"database/sql"
	"log"
	"mailinglist/grpcapi"
	"mailinglist/jsonapi"
	"mailinglist/mdb"
	"os"
	"os/signal"

	"github.com/alexflint/go-arg"
)

var args struct {
	DbPath   string `arg:"env:MAILING_LIST_DB"`
	BindJson string `arg:"env:MAILING_LIST_BIND_PORT"`
	BindGrpc string `arg:"env:MAILING_LIST_GRPC_BIND_PORT"`
}

func main() {
	arg.MustParse(&args)

	// Default values
	if args.DbPath == "" {
		args.DbPath = "list.db"
	}
	if args.BindJson == "" {
		args.BindJson = ":9091"
	}
	if args.BindGrpc == "" {
		args.BindGrpc = ":9092"
	}

	log.Printf("using db path %v and bind address %v\n", args.DbPath, args.BindJson)

	db, err := sql.Open("sqlite3", args.DbPath)
	if err != nil {
		log.Fatalf("Error opening sqlite db : %v\n", err)
	}
	defer db.Close()

	mdb.TryCreate(db)

	jsonServer := jsonapi.Serve(db, args.BindJson)
	defer func() {
		log.Println("HTTP Server graceful stop...")
		jsonapi.Shutdown(jsonServer)
	}()

	grpcServer := grpcapi.Serve(db, args.BindGrpc)
	defer func() {
		log.Println("gRPC Server graceful stop...")
		grpcServer.GracefulStop()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Kill)
	signal.Notify(sigChan, os.Interrupt)

	sig := <-sigChan
	log.Printf("Received terminal signal %v, Graceful shutdown\n", sig)

}
