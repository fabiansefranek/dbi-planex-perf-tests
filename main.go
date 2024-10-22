package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	/* POSTGRES */
	postgresConnectionString, postgresContainer, err := StartPostgres()
	if err != nil {
		panic(err)
	}
	defer postgresContainer.Terminate(context.Background())

	postgresConn, err := ConnectPostgres(postgresConnectionString)
	if err != nil {
		panic(err)
	}
	defer postgresConn.Close(context.Background())

	var result int

	err = postgresConn.QueryRow(context.Background(), "SELECT 1").Scan(&result)
	if err != nil {
		panic(err)
	}

	err = InitializePostgres(postgresConn)
	if err != nil {
		panic(err)
	}

	/* MONGODB */

	mongodbConnectionString, mongodbContainer, err := StartMongoDB()
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(mongodbContainer); err != nil {
			log.Printf("failed to terminate container")
			panic(err)
			
		}
	}()

	mongodbConn, err := ConnectMongoDB(mongodbConnectionString)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := mongodbConn.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()

	err = InitializeMongoDB(mongodbConn)
	if err != nil {
		panic(err)
	}

	time.Sleep(1 * time.Hour) 
}

/* POSTGRES */

func StartPostgres() (connectionString string, container *postgres.PostgresContainer, err error) {
    ctx := context.Background()
    postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		postgres.BasicWaitStrategies(),
	  )

	if err != nil {
		return "", nil, err
	}

	conn, err := postgresContainer.ConnectionString(ctx)

	if err != nil {
		return "", nil, err
	}

	return conn, postgresContainer, nil
}

func ConnectPostgres(connectionString string) (conn *pgx.Conn, err error) {
	conn, err = pgx.Connect(context.Background(), connectionString)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func InitializePostgres(conn *pgx.Conn) (err error) {
	_, err = conn.Exec(context.Background(), 
			`CREATE TABLE users (
				id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				username VARCHAR(255) NOT NULL,
				password VARCHAR(255) NOT NULL,
				first_name VARCHAR(255) NOT NULL,
				last_name VARCHAR(255) NOT NULL
			);

			CREATE TABLE projects (
				id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				identifier VARCHAR(48) NOT NULL,
				invite_code VARCHAR(128) NOT NULL,
				sprint_duration INT NOT NULL,
				owner_id INT NOT NULL,
				CONSTRAINT fk_owner FOREIGN KEY(owner_id) REFERENCES users(id)
			);

			CREATE TABLE sprints (
				id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				project_id INT NOT NULL,
				start_date DATE NOT NULL,
				end_date DATE NOT NULL,
				CONSTRAINT fk_project FOREIGN KEY(project_id) REFERENCES projects(id)
			);
`)
	if err != nil {
		return err
	}
	return nil
}

/* MONGODB */

func StartMongoDB() (connectionString string, container *mongodb.MongoDBContainer, err error) {
	ctx := context.Background()
	mongodbContainer, err := mongodb.Run(ctx, "mongo:6", mongodb.WithUsername("user"), mongodb.WithPassword("password"))

	if err != nil {
		return "", nil, err
	}

	connectionString, err = mongodbContainer.ConnectionString(ctx)
	if err != nil {
		return "", nil, err
	}

	return connectionString, mongodbContainer, nil
}

func ConnectMongoDB(connectionString string) (client *mongo.Client, err error) {
	client, err = mongo.Connect(context.Background(),options.Client().ApplyURI(connectionString))
	if err != nil {
		return nil, err
	}

	return client, nil
}

func InitializeMongoDB(client *mongo.Client) (err error) {
	ctx := context.Background()
	_, err = client.Database("test").Collection("users").InsertOne(ctx, bson.M{"username": "test", "password": "test", "first_name": "test", "last_name": "test"})
	if err != nil {
		return err
	}

	return nil
}
