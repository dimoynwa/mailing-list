package jsonapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"mailinglist/mdb"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
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

	if len(dataJson) == 0 {
		writer.WriteHeader(http.StatusNoContent)
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

func extractIdFromRequest(request *http.Request) (int64, error) {
	vars := mux.Vars(request)
	idStr := vars["id"]

	return strconv.ParseInt(idStr, 10, 64)
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

		email := request.URL.Query().Get("email")

		returnJson(writer, func() (interface{}, error) {
			log.Printf("JSON Get email: %v\n", email)
			return mdb.GetEmail(db, email)
		})
	})
}

func GetBatchEmail(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

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
		id, err := extractIdFromRequest(request)
		if err != nil {
			returnErr(writer, err, http.StatusBadRequest)
			return
		}

		entry := &mdb.EmailEntry{}
		fromJson(request.Body, entry)

		if err := mdb.UpdateEmail(db, *entry, id); err != nil {
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
		id, err := extractIdFromRequest(request)
		if err != nil {
			returnErr(writer, err, http.StatusBadRequest)
			return
		}

		if err = mdb.DeleteEmail(db, id); err != nil {
			returnErr(writer, err, http.StatusBadRequest)
			return
		}

		returnJson(writer, func() (interface{}, error) {
			log.Printf("JSON Delete email for ID: %v\n", id)
			return "", nil
		})
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[request] -> [%s] %s\n", r.Method, r.RequestURI)
		lrw := negroni.NewResponseWriter(w)
		defer func() {
			log.Printf("[response] -> [%s] [%d]\n", r.RequestURI, lrw.Status())
		}()
		next.ServeHTTP(lrw, r)
	})
}

func Serve(db *sql.DB, bind string) *http.Server {
	router := mux.NewRouter().StrictSlash(true)

	api := router.PathPrefix("/email").Subrouter()
	api.Use(loggingMiddleware)
	api.Handle("", GetEmail(db)).Methods(http.MethodGet)
	api.Handle("", CreateEmail(db)).Methods(http.MethodPost)
	api.Handle("/{id}", UpdateEmail(db)).Methods(http.MethodPut)
	api.Handle("/{id}", DeleteEmail(db)).Methods(http.MethodDelete)

	api.Handle("/batch", GetBatchEmail(db)).Methods(http.MethodGet)

	log.Printf("JSON API serve and listening on %v\n", bind)

	serv := &http.Server{
		Addr:         bind,
		Handler:      router,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	go func() {
		log.Printf("Starting server on port %v...\n ", serv.Addr)
		if err := serv.ListenAndServe(); err != nil {
			log.Fatalf("error starting the server: %v", err)
		}
	}()

	return serv

}

func Shutdown(serv *http.Server) {
	tc, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	serv.Shutdown(tc)
}
