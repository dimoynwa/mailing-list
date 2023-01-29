package main

import (
	"context"
	"fmt"
	"log"
	"mailinglist/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/alexflint/go-arg"
)

func logResponse(res *proto.EmailResponse, err error) {
	if err != nil {
		log.Fatalf("	error: %v\n", err)
	}

	if res.EmailEntry == nil {
		log.Println("	email not found")
	} else {
		log.Printf("	response: %v\n", res.EmailEntry)
	}
}

func createEmail(pb proto.MailingListServiceClient, addr string) *proto.EmailEntry {
	log.Printf("gRPC Client -> create email : %v\n", addr)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	res, err := pb.CreateEmail(ctx, &proto.CreateEmailRequest{EmailAddr: addr})
	logResponse(res, err)
	return res.EmailEntry
}

func updateEmail(pb proto.MailingListServiceClient, emailEntry proto.EmailEntry) *proto.EmailEntry {
	log.Printf("gRPC Client -> update email : %v\n", emailEntry.Email)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	res, err := pb.UpdateEmail(ctx, &proto.UpdateEmailRequest{EmailEntry: &emailEntry})
	logResponse(res, err)
	return res.EmailEntry
}

func deleteEmail(pb proto.MailingListServiceClient, addr string) *proto.EmailEntry {
	log.Printf("gRPC Client -> delete email : %v\n", addr)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	res, err := pb.DeleteEmail(ctx, &proto.DeleteEmailRequest{EmailAddr: addr})
	logResponse(res, err)
	return res.EmailEntry
}

func getEmail(pb proto.MailingListServiceClient, addr string) *proto.EmailEntry {
	log.Printf("gRPC Client -> get email : %v\n", addr)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	res, err := pb.GetEmail(ctx, &proto.GetEmailRequest{EmailAddr: addr})
	logResponse(res, err)
	return res.EmailEntry
}

func getEmailBatch(pb proto.MailingListServiceClient, page int32, count int32) []*proto.EmailEntry {
	log.Printf("gRPC Client -> get email batch : Page[%v] Count[%v]\n", page, count)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	res, err := pb.GetEmailBatch(ctx, &proto.GetEmailBatchRequest{Page: page, Count: count})
	if err != nil {
		log.Fatalf("	error: %v\n", err)
	}
	if len(res.EmailEntries) == 0 {
		log.Println("	no email entries found")
	} else {
		log.Printf("\tEMailEntries : [\n")
		for _, entry := range res.EmailEntries {
			log.Printf("\t\t%v\n", entry)
		}
		log.Printf("\t]\n")
	}
	return res.EmailEntries
}

var args struct {
	GrpcAddr string `arg:"env:MAILING_LIST_GRPC_ADDR"`
}

func main() {
	arg.MustParse(&args)

	if args.GrpcAddr == "" {
		args.GrpcAddr = ":9092"
	}

	conn, err := grpc.Dial(args.GrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error connecting to gRPC client at %v : %v\n", args.GrpcAddr, err)
	}
	defer conn.Close()

	client := proto.NewMailingListServiceClient(conn)

	emailAddr := fmt.Sprintf("dimodrangov%d@gmail.com", time.Now().Nanosecond())

	// Create email
	newEmail := createEmail(client, emailAddr)

	// Update email
	newEmail.ConfirmedAt = 10000
	updateEmail(client, *newEmail)

	// Get Email
	getEmail(client, newEmail.Email)

	// Delete email
	deleteEmail(client, newEmail.Email)

	// Get email batch
	getEmailBatch(client, 1, 5)
}
