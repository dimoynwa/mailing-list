package mdb

import (
	"database/sql"
	"log"
	"time"

	"github.com/mattn/go-sqlite3"
)

type EmailEntry struct {
	Id          int64
	Email       string
	ConfirmedAt *time.Time
	OptOut      bool
}

func TryCreate(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE emails (
			id 				INTEGER PRIMARY KEY,
			email   		TEXT UNIQUE,
			confirmed_at  	INTEGER,
			opt_out			INTEGER
		);
	`)

	if err != nil {
		if sqlerr, ok := err.(sqlite3.Error); ok {
			// Code 1 means that table already exists
			if sqlerr.Code != 1 {
				log.Fatalf("cannot create db: %v", sqlerr)
			}
		} else {
			log.Fatalf("unexpected error creating DB: %v", err)
		}
	}
}

func emailEntryFromRow(row *sql.Rows) (*EmailEntry, error) {
	var (
		id          int64
		email       string
		confirmedAt int64
		optOut      bool
	)
	err := row.Scan(&id, &email, &confirmedAt, &optOut)
	if err != nil {
		return nil, err
	}

	t := time.Unix(confirmedAt, 0)
	return &EmailEntry{
		Id:          id,
		Email:       email,
		ConfirmedAt: &t,
		OptOut:      optOut,
	}, nil
}

func CreateEmail(db *sql.DB, email string) error {
	_, err := db.Exec(`
		INSERT INTO emails (email, confirmed_at, opt_out)
		VALUES (?, 0, false)
	`, email)

	if err != nil {
		log.Printf("Error creating email for %v\n", email)
		return err
	}
	return nil
}

func GetEmail(db *sql.DB, email string) (*EmailEntry, error) {
	rows, err := db.Query(`
		SELECT id, email, confirmed_at, opt_out 
		FROM emails where email = ?`, email)

	if err != nil {
		log.Printf("Error getting emailEntry for %v: %v\n", email, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		return emailEntryFromRow(rows)
	}
	return nil, nil
}

func UpdateEmail(db *sql.DB, emailEntry EmailEntry) error {
	t := emailEntry.ConfirmedAt.Unix()

	_, err := db.Exec(`
		INSERT INTO emails (email, confirmed_at, opt_out)
		VALUES (?, ?, ?)
		ON CONFLICT (email) DO UPDATE SET
			confirmed_at=?
			opt_out=?
	`, emailEntry.Email, t, emailEntry.OptOut, t, emailEntry.OptOut)

	if err != nil {
		log.Printf("Error upserting email for entry %v: %v\n", emailEntry, err)
		return err
	}

	return nil
}

func DeleteEmail(db *sql.DB, email string) error {
	_, err := db.Exec(`
		UPDATE emails SET opt_out=true WHERE email = ?
	`, email)

	if err != nil {
		log.Printf("Error deleting email %v: %v\n", email, err)
		return err
	}
	return nil
}

type GetBatchEmailQueryParams struct {
	Page, Count int
}

func GetEmailBatch(db *sql.DB, params GetBatchEmailQueryParams) ([]*EmailEntry, error) {
	var empty []*EmailEntry

	rows, err := db.Query(`
		SELECT id, email, confirmed_at, opt_out FROM email 
		WHERE opt_out=false ORDER BY id ASC
		LIMIT ? OFFSET ?
	`, params.Count, (params.Page-1)*params.Count)

	if err != nil {
		log.Printf("Error getting batch emails: %v\n", err)
		return empty, err
	}

	defer rows.Close()

	emails := make([]*EmailEntry, 0, params.Count)

	for rows.Next() {
		email, err := emailEntryFromRow(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, nil
}
