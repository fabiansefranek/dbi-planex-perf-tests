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
)

func AddSprint(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	name := r.PostFormValue("name")
	startDate := r.PostFormValue("start_date")
	endDate := r.PostFormValue("end_date")
	projectId := r.PostFormValue("project_id")

    query := `insert into sprints (name, start_date, end_date, project_id) values (@name, @start_date, @end_date, @project_id)`
    args := pgx.NamedArgs{
        "name" : name,
        "start_date": startDate,
        "end_date": endDate,
		"project_id": projectId,
    }
    _, err = db.PostgresConn.Exec(context.Background(), query, args)

    if err != nil {
        panic(err)
    }

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func GetSprints() (sprints [][]string) {
	rows, err := db.PostgresConn.Query(context.Background(), `select * from sprints`)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		var startDate int64
		var endDate int64
		var projectId int
		err = rows.Scan(&id, &name, &startDate, &endDate, &projectId)
		if err != nil {
			panic(err)
		}

		sprints = append(sprints, []string{strconv.Itoa(id), name, strconv.FormatInt(startDate, 10), strconv.FormatInt(endDate, 10), strconv.Itoa(projectId)})
	}

	return sprints
}

func GetSprint(id int) (sprint models.Sprint) {
	err := db.PostgresConn.QueryRow(context.Background(), `select * from sprints where id = $1`, id).Scan(&sprint.Id, &sprint.Name, &sprint.StartDate, &sprint.EndDate, &sprint.ProjectId)
	if err != nil {
		panic(err)
	}

	return sprint
}

func DeleteSprint(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	query := `delete from sprints where id = @id`
	args := pgx.NamedArgs{
		"id" : id,
	}
	_, err = db.PostgresConn.Exec(context.Background(), query, args)

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func UpdateSprint(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	query := `update sprints set name = @name, start_date = @start_date, end_date = @end_date where id = @id`
	args := pgx.NamedArgs{
		"id" : id,
		"name": r.PostFormValue("name"),
		"start_date": r.PostFormValue("start_date"),
		"end_date": r.PostFormValue("end_date"),
	}

	_, err = db.PostgresConn.Exec(context.Background(), query, args)

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}	

/* MONGODB */

func AddMongoSprint(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	name := r.PostFormValue("name")
	startDate := r.PostFormValue("start_date")
	endDate := r.PostFormValue("end_date")
	projectId := r.PostFormValue("project_id")

	startDateInt, err := strconv.Atoi(startDate)
	if err != nil {
		panic(err)
	}

	endDateInt, err := strconv.Atoi(endDate)
	if err != nil {
		panic(err)
	}

    _, err = db.MongoConn.Database("test").Collection("sprints").InsertOne(context.Background(), bson.M{
        "name": name,
        "start_date": startDateInt,
        "end_date": endDateInt,
		"project_id": projectId,
    })

    if err != nil {
        panic(err)
    }

	http.Redirect(w, r, "/mongo", http.StatusTemporaryRedirect)
}

func GetMongoSprints() (sprints [][]string) {
	cursor, err := db.MongoConn.Database("test").Collection("sprints").Find(context.Background(), bson.M{})
	if err != nil {
		panic(err)
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var sprint models.Sprint
		err = cursor.Decode(&sprint)
		if err != nil {
			panic(err)
		}

		sprints = append(sprints, []string{sprint.MongoId.Hex(), sprint.Name, strconv.FormatInt(sprint.StartDate, 10), strconv.FormatInt(sprint.EndDate, 10), sprint.MongoProjectId.Hex()})
	}

	return sprints
}

func GetMongoSprint(id string) (sprint models.Sprint) {
	ctx := context.Background()
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Println("FAiled to parse ObjectID")
		panic(err)
	}

	result := db.MongoConn.Database("test").Collection("sprints").FindOne(ctx, bson.M{"_id": oid})

	err = result.Decode(&sprint)
	if err != nil {
		panic(err)
	}

	return sprint
}

func DeleteMongoSprint(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		panic(err)
	}

	_, err = db.MongoConn.Database("test").Collection("sprints").DeleteOne(context.Background(), bson.M{"_id": oid})
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/mongo", http.StatusTemporaryRedirect)
}

func UpdateMongoSprint(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		panic(err)
	}

	_, err = db.MongoConn.Database("test").Collection("sprints").UpdateOne(context.Background(), bson.M{"_id": oid}, bson.M{"$set": bson.M{
		"name": r.PostFormValue("name"),
		"start_date": r.PostFormValue("start_date"),
		"end_date": r.PostFormValue("end_date"),
	}})

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/mongo", http.StatusTemporaryRedirect)
}	