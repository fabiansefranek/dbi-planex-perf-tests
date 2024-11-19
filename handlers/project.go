package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fabiansefranek/dbi-perf-tests/db"
	"github.com/fabiansefranek/dbi-perf-tests/models"
	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func AddProject(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	name := r.PostFormValue("name")
	identifier := r.PostFormValue("identifier")
	inviteCode := r.PostFormValue("invite_code")
	sprintDuration := r.PostFormValue("sprint_duration")
	ownerId := r.PostFormValue("owner_id")

    query := `insert into projects (name, identifier, invite_code, sprint_duration, owner_id) values (@name, @identifier, @invite_code, @sprint_duration, @owner_id)`
    args := pgx.NamedArgs{
        "name" : name,
		"identifier": identifier,
		"invite_code": inviteCode,
		"sprint_duration": sprintDuration,
		"owner_id": ownerId,
    }
    _, err = db.PostgresConn.Exec(context.Background(), query, args)

    if err != nil {
        panic(err)
    }

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func GetProjects(nameSearch string) (projects [][]string) {
	var query string

	if nameSearch != "" {
		query = `select * from projects where name like $1`
	} else {
		query = `select * from projects`
	}

	var err error
	var rows pgx.Rows

	if nameSearch != "" {
		rows, err = db.PostgresConn.Query(context.Background(), query, nameSearch)
		if err != nil {
			panic(err)
		}
	} else {
		rows, err = db.PostgresConn.Query(context.Background(), query)
		if err != nil {
			panic(err)
		}
	}

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		var id int
		var name string 
		var identifier string
		var inviteCode string
		var sprintDuration int
		var ownerId int
		err = rows.Scan(&id, &name, &identifier, &inviteCode, &sprintDuration, &ownerId)

		if err != nil {
			panic(err)
		}

		projects = append(projects, []string{strconv.Itoa(id), name, identifier, inviteCode, strconv.Itoa(sprintDuration), strconv.Itoa(ownerId)})
	}

	return projects
}

func GetProject(id int) (project models.Project) {
	err := db.PostgresConn.QueryRow(context.Background(), `select * from projects where id = $1`, id).Scan(&project.Id, &project.Name, &project.Identifier, &project.InviteCode, &project.SprintDuration, &project.OwnerId)
	if err != nil {
		panic(err)
	}

	return project
}

func DeleteProject(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	query := `delete from projects where id = @id`
	args := pgx.NamedArgs{
		"id" : id,
	}
	_, err = db.PostgresConn.Exec(context.Background(), query, args)

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func UpdateProject(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	query := `update projects set name = @name, identifier = @identifier, invite_code = @invite_code, sprint_duration = @sprint_duration, owner_id = @owner_id where id = @id`
	args := pgx.NamedArgs{
		"id": id,
		"name": r.PostFormValue("name"),
		"identifier": r.PostFormValue("identifier"),
		"invite_code": r.PostFormValue("invite_code"),
		"sprint_duration": r.PostFormValue("sprint_duration"),
		"owner_id": r.PostFormValue("owner_id"),
	}

	_, err = db.PostgresConn.Exec(context.Background(), query, args)

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}	

/* MONGODB */


func AddMongoProject(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	name := r.PostFormValue("name")
	identifier := r.PostFormValue("identifier")
	inviteCode := r.PostFormValue("invite_code")
	sprintDuration := r.PostFormValue("sprint_duration")
	ownerId := r.PostFormValue("owner_id")

	duration, err := strconv.Atoi(sprintDuration)
	if err != nil {
		panic(err)
	}
	
	_, err = db.MongoConn.Database("test").Collection("projects").InsertOne(context.Background(), bson.M{
		"name": name,
		"identifier": identifier,
		"invite_code": inviteCode,
		"sprint_duration": duration,
		"owner_id": ownerId,
	})

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/mongo", http.StatusTemporaryRedirect)
}

func GetMongoProjects(nameSearch string) (projects [][]string) {
	var cursor *mongo.Cursor
	var err error

	if nameSearch != "" {
		cursor, err = db.MongoConn.Database("test").Collection("projects").Find(context.Background(), bson.M{"name": bson.M{"$regex": nameSearch}})
		if err != nil {
			panic(err)
		}
	} else {
		cursor, err = db.MongoConn.Database("test").Collection("projects").Find(context.Background(), bson.M{})
		if err != nil {
			panic(err)
		}
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var project models.Project
		err = cursor.Decode(&project)
		if err != nil {
			panic(err)
		}

		projects = append(projects, []string{project.MongoId.Hex(), project.Name, project.Identifier, project.InviteCode, strconv.Itoa(project.SprintDuration), project.MongoOwnerId.Hex()})
	}

	return projects
}

func GetMongoProject(id string) (project models.Project) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Println("FAiled to parse ObjectID")
		panic(err)
	}

	result := db.MongoConn.Database("test").Collection("projects").FindOne(context.Background(), bson.M{"_id": oid})

	err = result.Decode(&project)
	if err != nil {
		panic(err)
	}

	return project
}

func DeleteMongoProject(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		panic(err)
	}

	_, err = db.MongoConn.Database("test").Collection("projects").DeleteOne(context.Background(), bson.M{"_id": oid})

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/mongo", http.StatusTemporaryRedirect)
}

func UpdateMongoProject(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		panic(err)
	}

	duration, err := strconv.Atoi(r.PostFormValue("sprint_duration"))
	if err != nil {
		panic(err)
	}

	_, err = db.MongoConn.Database("test").Collection("projects").UpdateOne(context.Background(), bson.M{"_id": oid}, bson.M{"$set": bson.M{
		"owner_id": r.PostFormValue("owner_id"),
		"name": r.PostFormValue("name"),
		"identifier": r.PostFormValue("identifier"),
		"invite_code": r.PostFormValue("invite_code"),
		"sprint_duration": duration,
	}})

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/mongo", http.StatusTemporaryRedirect)
}	