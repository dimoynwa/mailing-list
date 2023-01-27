package jsonapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"mailinglist/mdb"
	"net/http"
	"strconv"
)

func setJsonHeader(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
}

func fromJson[T any](r io.Reader, dest T) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)

	json.Unmarshal(buf.Bytes(), &dest)
}

func returnJson[T any](writer http.ResponseWriter, withData func() (T, error)) {
	setJsonHeader(writer)
	data, err := withData()

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		errJson, err := json.Marshal(&err)
		if err != nil {
			log.Println(err)
		}
		writer.Write(errJson)
		return
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Write(dataJson)
}

func returnErr(writer http.ResponseWriter, err error, code int) {
	returnJson(writer, func() (interface{}, error) {
		errorMessage := struct {
			Err string
		}{
			Err: err.Error(),
		}

		writer.WriteHeader(code)
		return errorMessage, nil
	})
}

func getPagingParams(request *http.Request) (*mdb.GetBatchEmailQueryParams, error) {
	pageParam := request.URL.Query().Get("page")
	countParam := request.URL.Query().Get("count")

	page := 0
	var err error
	if pageParam != "" {
		page, err = strconv.Atoi(pageParam)
		if err != nil {
			return nil, err
		}
	}

	count := 5 // Default value
	if countParam != "" {
		count, err = strconv.Atoi(countParam)
		if err != nil {
			return nil, err
		}
	}

	return &mdb.GetBatchEmailQueryParams{Page: page, Count: count}, nil
}

func CreateEmail(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		entry := &mdb.EmailEntry{}
		fromJson(request.Body, entry)

		if err := mdb.CreateEmail(db, entry.Email); err != nil {
			returnErr(writer, err, http.StatusBadRequest)
			return
		}

		returnJson(writer, func() (interface{}, error) {
			log.Printf("JSON Create email: %v\n", entry.Email)
			return mdb.GetEmail(db, entry.Email)
		})
	})
}

func GetEmail(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		email := request.URL.Query().Get("email")

		returnJson(writer, func() (interface{}, error) {
			log.Printf("JSON Get email: %v\n", email)
			return mdb.GetEmail(db, email)
		})
	})
}

func GetBatchEmail(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		params, err := getPagingParams(request)

		if err != nil {
			returnErr(writer, err, http.StatusBadRequest)
		}

		returnJson(writer, func() (interface{}, error) {
			log.Printf("JSON Get batch email: %v\n", params)
			return mdb.GetEmailBatch(db, *params)
		})
	})
}

func UpdateEmail(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPut {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		entry := &mdb.EmailEntry{}
		fromJson(request.Body, entry)

		if err := mdb.UpdateEmail(db, *entry); err != nil {
			returnErr(writer, err, http.StatusBadRequest)
			return
		}

		returnJson(writer, func() (interface{}, error) {
			log.Printf("JSON Update email: %v\n", entry.Email)
			return mdb.GetEmail(db, entry.Email)
		})
	})
}

func DeleteEmail(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		entry := &mdb.EmailEntry{}
		fromJson(request.Body, entry)

		if err := mdb.DeleteEmail(db, entry.Email); err != nil {
			returnErr(writer, err, http.StatusBadRequest)
			return
		}

		returnJson(writer, func() (interface{}, error) {
			log.Printf("JSON Delete email: %v\n", entry.Email)
			return mdb.GetEmail(db, entry.Email)
		})
	})
}

func Serve(db *sql.DB, bind string) {
	http.Handle("/email/create", CreateEmail(db))
	http.Handle("/email/get", GetEmail(db))
	http.Handle("/email/get_batch", GetBatchEmail(db))
	http.Handle("/email/update", UpdateEmail(db))
	http.Handle("/email/delete", DeleteEmail(db))
	log.Printf("JSON API serve and listening on %v\n", bind)

	err := http.ListenAndServe(bind, nil)
	if err != nil {
		log.Fatalf("Error starting server at %v\n", bind)
	}
}
