package grpcapi

import (
	"context"
	"database/sql"
	"log"
	"mailinglist/mdb"
	"mailinglist/proto"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
)

type MailService struct {
	proto.UnimplementedMailingListServiceServer
	db     *sql.DB
	logger *log.Logger
}

func Serve(db *sql.DB, bind string) *grpc.Server {
	logger := log.New(os.Stdout, "gRPC mail service -> ", log.Ldate|log.Ltime)

	listener, err := net.Listen("tcp", bind)
	if err != nil {
		logger.Fatalf("gRPC error, failed to start : %v\n", err)
	}

	grpcServer := grpc.NewServer()

	mailService := MailService{
		db:     db,
		logger: logger,
	}

	proto.RegisterMailingListServiceServer(grpcServer, &mailService)

	logger.Printf("gRPC API service starting on %v\n", bind)

	go func() {
		log.Printf("Starting gRPC server on port %v...\n ", bind)
		if err = grpcServer.Serve(listener); err != nil {
			logger.Fatalf("gRPC error: %v\n", err)
		}
	}()

	return grpcServer
}

func pbEntryToMdb(pb *proto.EmailEntry) *mdb.EmailEntry {
	t := time.Unix(pb.ConfirmedAt, 0)

	mdbEntry := mdb.EmailEntry{Id: pb.Id, Email: pb.Email, ConfirmedAt: &t, OptOut: pb.OptOut}
	return &mdbEntry
}

func mdbEntryToPb(mdbEntry *mdb.EmailEntry) *proto.EmailEntry {
	return &proto.EmailEntry{
		Id:          mdbEntry.Id,
		Email:       mdbEntry.Email,
		ConfirmedAt: mdbEntry.ConfirmedAt.Unix(),
		OptOut:      mdbEntry.OptOut,
	}
}

func emailResponse(db *sql.DB, email string) (*proto.EmailResponse, error) {
	entry, err := mdb.GetEmail(db, email)
	if err != nil {
		return &proto.EmailResponse{}, err
	}
	if entry == nil {
		return &proto.EmailResponse{}, nil
	}

	res := mdbEntryToPb(entry)
	return &proto.EmailResponse{EmailEntry: res}, nil
}

func (s *MailService) CreateEmail(ctx context.Context, r *proto.CreateEmailRequest) (*proto.EmailResponse, error) {
	s.logger.Printf("Create email: %v\n", r.EmailAddr)

	if err := mdb.CreateEmail(s.db, r.EmailAddr); err != nil {
		return &proto.EmailResponse{}, err
	}

	return emailResponse(s.db, r.EmailAddr)
}

func (s *MailService) UpdateEmail(ctx context.Context, r *proto.UpdateEmailRequest) (*proto.EmailResponse, error) {
	s.logger.Printf("Update email for %v\n", r.EmailEntry)

	mdbEntry := pbEntryToMdb(r.EmailEntry)

	if err := mdb.UpsertEmail(s.db, *mdbEntry); err != nil {
		return &proto.EmailResponse{}, err
	}

	return emailResponse(s.db, mdbEntry.Email)
}

func (s *MailService) DeleteEmail(ctx context.Context, r *proto.DeleteEmailRequest) (*proto.EmailResponse, error) {
	s.logger.Printf("Delete email for %v\n", r.EmailAddr)

	if err := mdb.DeleteEmailByEmail(s.db, r.EmailAddr); err != nil {
		return &proto.EmailResponse{}, err
	}
	return emailResponse(s.db, r.EmailAddr)
}

func (s *MailService) GetEmail(ctx context.Context, r *proto.GetEmailRequest) (*proto.EmailResponse, error) {
	s.logger.Printf("Get email: %v\n", r.EmailAddr)
	return emailResponse(s.db, r.EmailAddr)
}

func (s *MailService) GetEmailBatch(ctx context.Context, r *proto.GetEmailBatchRequest) (*proto.GetEmailBatchResponse, error) {
	s.logger.Printf("GetEmailBatch: count %v, page: %v\n", r.Count, r.Page)

	params := mdb.GetBatchEmailQueryParams{
		Count: int(r.Count),
		Page:  int(r.Page),
	}

	entries, err := mdb.GetEmailBatch(s.db, params)
	if err != nil {
		return &proto.GetEmailBatchResponse{}, err
	}

	pbEntries := make([]*proto.EmailEntry, 0, len(entries))
	for _, entry := range entries {
		pbEntries = append(pbEntries, mdbEntryToPb(entry))
	}
	return &proto.GetEmailBatchResponse{EmailEntries: pbEntries}, nil
}
