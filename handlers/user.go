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

func AddUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	username := r.PostFormValue("username")
	firstname := r.PostFormValue("firstName")
	lastname := r.PostFormValue("lastName")

    query := `insert into users (username, first_name, last_name) values (@username, @firstName, @lastName)`
    args := pgx.NamedArgs{
        "username" : username,
        "firstName": firstname,
        "lastName": lastname,
    }
    _, err = db.PostgresConn.Exec(context.Background(), query, args)

    if err != nil {
        panic(err)
    }

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func GetUsers() (users [][]string) {
	rows, err := db.PostgresConn.Query(context.Background(), `select * from users`)

	if err != nil {
		panic(err)
	}

	defer rows.Close()

	for rows.Next() {
		var id int
		var username string
		var firstname string
		var lastname string
		err = rows.Scan(&id, &username, &firstname, &lastname)
		if err != nil {
			panic(err)
		}

		fmt.Println(id, username, firstname, lastname)

		users = append(users, []string{strconv.Itoa(id), username, firstname, lastname})
	}

	return users
}

func GetUser(id int) (user models.User) {
	err := db.PostgresConn.QueryRow(context.Background(), `select * from users where id = $1`, id).Scan(&user.Id, &user.Username, &user.FirstName, &user.LastName)
	if err != nil {
		panic(err)
	}

	return user
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	query := `delete from users where id = @id`
	args := pgx.NamedArgs{
		"id" : id,
	}
	_, err = db.PostgresConn.Exec(context.Background(), query, args)

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	query := `update users set username = @username, first_name = @firstname, last_name = @lastname where id = @id`
	args := pgx.NamedArgs{
		"id" : id,
		"username": r.PostFormValue("username"),
		"firstname": r.PostFormValue("firstName"),
		"lastname": r.PostFormValue("lastName"),
	}
	_, err = db.PostgresConn.Exec(context.Background(), query, args)

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}	

/* MONGODB */

func AddMongoUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	username := r.PostFormValue("username")
	firstname := r.PostFormValue("firstName")
	lastname := r.PostFormValue("lastName")

	_, err = db.MongoConn.Database("test").Collection("users").InsertOne(context.Background(), bson.M{
		"username": username,
		"first_name": firstname,
		"last_name": lastname,
	})

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/mongo", http.StatusTemporaryRedirect)
}

func GetMongoUsers() (users [][]string) {
	cursor, err := db.MongoConn.Database("test").Collection("users").Find(context.Background(), bson.M{})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var user models.User
		err = cursor.Decode(&user)
		if err != nil {
			panic(err)
		}

		users = append(users, []string{user.MongoId.Hex(), user.Username, user.FirstName, user.LastName})
	}

	return users
}

func GetMongoUser(id string) (user models.User) {
	ctx := context.Background()
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		fmt.Println("FAiled to parse ObjectID")
		panic(err)
	}

	result := db.MongoConn.Database("test").Collection("users").FindOne(ctx, bson.M{"_id": oid})

	err = result.Decode(&user)
	if err != nil {
		panic(err)
	}

	return user
}

func DeleteMongoUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		panic(err)
	}

	_, err = db.MongoConn.Database("test").Collection("users").DeleteOne(context.Background(), bson.M{"_id": oid})

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/mongo", http.StatusTemporaryRedirect)
}

func UpdateMongoUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm() 
	if err != nil{
		   panic(err)
	}
	
	id := r.PostFormValue("id")

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		panic(err)
	}

	_, err = db.MongoConn.Database("test").Collection("users").UpdateOne(context.Background(), bson.M{"_id": oid}, bson.M{"$set": bson.M{
		"username": r.PostFormValue("username"),
		"first_name": r.PostFormValue("firstName"),
		"last_name": r.PostFormValue("lastName"),
	}})

	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/mongo", http.StatusTemporaryRedirect)
}	