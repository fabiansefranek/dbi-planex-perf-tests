package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/fabiansefranek/dbi-perf-tests/handlers"
	"github.com/fabiansefranek/dbi-perf-tests/views"
)

func StartServer() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	/* POSTGRES */

	http.Handle("/", templ.Handler(views.PostgresIndex()))

	http.Handle("POST /postgres/users", http.HandlerFunc(handlers.AddUser))
	http.Handle("POST /postgres/users/delete", http.HandlerFunc(handlers.DeleteUser))
	http.Handle("POST /postgres/users/update", http.HandlerFunc(handlers.UpdateUser))
	http.HandleFunc("GET /postgres/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		idInt, err := strconv.Atoi(id)
		if err != nil {
			panic(err)
		}

		views.User("postgres", idInt).Render(r.Context(), w)
	})

	http.Handle("POST /postgres/sprints", http.HandlerFunc(handlers.AddSprint))
	http.Handle("POST /postgres/sprints/delete", http.HandlerFunc(handlers.DeleteSprint))
	http.Handle("POST /postgres/sprints/update", http.HandlerFunc(handlers.UpdateSprint))
	http.HandleFunc("GET /postgres/sprints/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		idInt, err := strconv.Atoi(id)
		if err != nil {
			panic(err)
		}

		views.Sprint("postgres", idInt).Render(r.Context(), w)
	})

	http.Handle("POST /postgres/projects", http.HandlerFunc(handlers.AddProject))
	http.Handle("POST /postgres/projects/delete", http.HandlerFunc(handlers.DeleteProject))
	http.Handle("POST /postgres/projects/update", http.HandlerFunc(handlers.UpdateProject))
	http.HandleFunc("GET /postgres/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		idInt, err := strconv.Atoi(id)
		if err != nil {
			panic(err)
		}

		views.Project("postgres", idInt).Render(r.Context(), w)
	})

	/* MONGODB */

	http.Handle("/mongo", templ.Handler(views.MongoIndex()))

	http.Handle("POST /mongo/users", http.HandlerFunc(handlers.AddMongoUser))
	http.Handle("POST /mongo/users/delete", http.HandlerFunc(handlers.DeleteMongoUser))
	http.Handle("POST /mongo/users/update", http.HandlerFunc(handlers.UpdateMongoUser))
	http.HandleFunc("GET /mongo/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		oid := r.PathValue("id")

		views.MongoUser("mongo", oid).Render(r.Context(), w)
	})

	http.Handle("POST /mongo/sprints", http.HandlerFunc(handlers.AddMongoSprint))
	http.Handle("POST /mongo/sprints/delete", http.HandlerFunc(handlers.DeleteMongoSprint))
	http.Handle("POST /mongo/sprints/update", http.HandlerFunc(handlers.UpdateMongoSprint))
	http.HandleFunc("GET /mongo/sprints/{id}", func(w http.ResponseWriter, r *http.Request) {
		oid := r.PathValue("id")

		views.MongoSprint("mongo", oid).Render(r.Context(), w)
	})

	http.Handle("POST /mongo/projects", http.HandlerFunc(handlers.AddMongoProject))
	http.Handle("POST /mongo/projects/delete", http.HandlerFunc(handlers.DeleteMongoProject))
	http.Handle("POST /mongo/projects/update", http.HandlerFunc(handlers.UpdateMongoProject))
	http.HandleFunc("GET /mongo/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		oid := r.PathValue("id")

		views.MongoProject("mongo", oid).Render(r.Context(), w)
	})

	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", nil)
}

