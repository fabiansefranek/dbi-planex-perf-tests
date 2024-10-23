package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
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

	err = InitializePostgres(postgresConn)
	if err != nil {
		panic(err)
	}

	mongoConnectionString, mongoContainer, err := StartMongoDB()
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(mongoContainer); err != nil {
			log.Printf("failed to terminate container")
			panic(err)
			
		}
	}()

	mongoConn, err := ConnectMongoDB(mongoConnectionString)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := mongoConn.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()

	err = InitializeMongoDB(mongoConn)
	if err != nil {
		panic(err)
	}

	projects := GenerateProjects(10000)
	tableRows := make([][]string, 0)

	/* PERFORMANCE TESTS */

	/* insert */

	postgresDuration, err := InsertPostgres(postgresConn, projects)
	if err != nil {
		panic(err)
	}

	mongoDuration, err := InsertMongo(mongoConn, projects)
	if err != nil {
		panic(err)
	}

	tableRows = append(tableRows, []string{"Insert", postgresDuration, mongoDuration})

	/* find */

	postgresDuration, err = FindPostgres(postgresConn)
	if err != nil {
		panic(err)
	}

	mongoDuration, err = FindMongo(mongoConn)
	if err != nil {
		panic(err)
	}

	tableRows = append(tableRows, []string{"Find", postgresDuration, mongoDuration})

	postgresDuration, err = FindPostgresWithFilter(postgresConn)
	if err != nil {
		panic(err)
	}

	mongoDuration, err = FindMongoWithFilter(mongoConn)
	if err != nil {
		panic(err)
	}

	tableRows = append(tableRows, []string{"Find with filter", postgresDuration, mongoDuration})

	postgresDuration, err = FindPostgresWithFilterAndProjection(postgresConn)
	if err != nil {
		panic(err)
	}

	mongoDuration, err = FindMongoWithFilterAndProjection(mongoConn)
	if err != nil {
		panic(err)
	}

	tableRows = append(tableRows, []string{"Find with filter and projection", postgresDuration, mongoDuration})

	postgresDuration, err = FindPostgresWithFilterAndProjectionAndSort(postgresConn)
	if err != nil {
		panic(err)
	}

	mongoDuration, err = FindMongoWithFilterAndProjectionAndSort(mongoConn)
	if err != nil {
		panic(err)
	}

	tableRows = append(tableRows, []string{"Find with filter and projection and sort", postgresDuration, mongoDuration})

	/* update */

	postgresDuration, err = UpdatePostgres(postgresConn)
	if err != nil {
		panic(err)
	}

	mongoDuration, err = UpdateMongo(mongoConn)
	if err != nil {
		panic(err)
	}

	tableRows = append(tableRows, []string{"Update", postgresDuration, mongoDuration})

	/* delete */

	postgresDuration, err = DeletePostgres(postgresConn)
	if err != nil {
		panic(err)
	}

	mongoDuration, err = DeleteMongo(mongoConn)
	if err != nil {
		panic(err)
	}
	
	tableRows = append(tableRows, []string{"Delete", postgresDuration, mongoDuration})

	/* TABLE */

	PrintTable(tableRows)

	time.Sleep(1 * time.Hour) 
}

/* TYPES */

type Project struct {
	Id int
	Name string
	Identifier string
	InviteCode string
	SprintDuration int
	Owner User
	Sprints []Sprint
}

type User struct {
	Id int
	Username string
	FirstName string
	LastName string
}

type Sprint struct {
	Id int
	Name string
	StartDate int64
	EndDate int64
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
				first_name VARCHAR(255) NOT NULL,
				last_name VARCHAR(255	) NOT NULL
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
				start_date INT NOT NULL,
				end_date INT NOT NULL,
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

/* SEEDER */

func RandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func RandomTimestamp() int64 {
	return rand.Int63n(time.Now().Unix() - 94608000) + 94608000
}

/* func GenerateStrings(arraySize int, stringLength int) []string {
	var result []string
	for i := 0; i < arraySize; i++ {
		result = append(result, RandomString(stringLength))
	}

	return result
}

func GenerateTimestamps(arraySize int) []string {
	var result []string
	for i := 0; i < arraySize; i++ {
		result = append(result, RandomTimestamp())
	}

	return result
} */

func GenerateProjects(arraySize int) []Project {
	var result []Project
	for i := 0; i < arraySize; i++ {
		result = append(result, Project{
			Name: RandomString(10),
			Identifier: RandomString(48),
			InviteCode: RandomString(128),
			SprintDuration: rand.Intn(100) + 1,
			Owner: User{
				Username: RandomString(10),
				FirstName: RandomString(10),
				LastName: RandomString(10),
			},
			Sprints: []Sprint{
				{
					Name: RandomString(10),
					StartDate: RandomTimestamp(),
					EndDate: RandomTimestamp(),
				},
			},
		})
	}
	return result
}

/* PERFORMANCE TESTS */

func InsertPostgres(conn *pgx.Conn, projects []Project) (duration string, err error) {
	now := time.Now()
	for _, project := range projects {
		var userId int
		err = conn.QueryRow(context.Background(), 
			`INSERT INTO users (username, first_name, last_name) VALUES ($1, $2, $3) RETURNING id;`,
			project.Owner.Username, project.Owner.FirstName, project.Owner.LastName).Scan(&userId)
		if err != nil {
			return "", err
		}

		var projectId int
		err = conn.QueryRow(context.Background(),
			`INSERT INTO projects (name, identifier, invite_code, sprint_duration, owner_id) VALUES ($1, $2, $3, $4, $5) RETURNING id;`,
			project.Name, project.Identifier, project.InviteCode, project.SprintDuration, userId).Scan(&projectId)
		if err != nil {
			return "", err
		}

		for _, sprint := range project.Sprints {
			var sprintId int
			err = conn.QueryRow(context.Background(),
				`INSERT INTO sprints (name, project_id, start_date, end_date) VALUES ($1, $2, $3, $4) RETURNING id;`,
				sprint.Name, projectId, sprint.StartDate, sprint.EndDate).Scan(&sprintId)
			if err != nil {
				return "", err
			}
		}
	}

	return time.Since(now).String(), nil
}

func FindPostgres(conn *pgx.Conn) (duration string, err error) {
	now := time.Now()	
	_, err = conn.Exec(context.Background(), `SELECT * FROM sprints INNER JOIN projects ON sprints.project_id = projects.id INNER JOIN users ON projects.owner_id = users.id;`)
	if err != nil {
		return "", err
	}

	return time.Since(now).String(), nil
}

func FindPostgresWithFilter(conn *pgx.Conn) (duration string, err error) {
	now := time.Now()	
	_, err = conn.Exec(context.Background(), `SELECT * FROM sprints INNER JOIN projects ON sprints.project_id = projects.id INNER JOIN users ON projects.owner_id = users.id WHERE projects.sprint_duration > 50;`)
	if err != nil {
		return "", err
	}

	return time.Since(now).String(), nil
}

func FindPostgresWithFilterAndProjection(conn *pgx.Conn) (duration string, err error) {
	now := time.Now()
	_, err = conn.Exec(context.Background(), `SELECT users.username, projects.name, sprints.name FROM sprints INNER JOIN projects ON sprints.project_id = projects.id INNER JOIN users ON projects.owner_id = users.id WHERE projects.sprint_duration > 50;`)
	if err != nil {
		return "", err
	}

	return time.Since(now).String(), nil
}

func FindPostgresWithFilterAndProjectionAndSort(conn *pgx.Conn) (duration string, err error) {
	now := time.Now()
	_, err = conn.Exec(context.Background(), `SELECT users.username, projects.name, sprints.name FROM sprints INNER JOIN projects ON sprints.project_id = projects.id INNER JOIN users ON projects.owner_id = users.id WHERE projects.sprint_duration > 50 ORDER BY sprints.start_date DESC;`)
	if err != nil {
		return "", err
	}

	return time.Since(now).String(), nil
}

func UpdatePostgres(conn *pgx.Conn) (duration string, err error) {
	now := time.Now()
	_, err = conn.Exec(context.Background(), `UPDATE sprints SET start_date = start_date + (60*60*24)`)
	if err != nil {
		return "", err
	}
	return time.Since(now).String(), nil
}

func DeletePostgres(conn *pgx.Conn) (duration string, err error) {
	now := time.Now()
	_, err = conn.Exec(context.Background(), `DELETE FROM sprints`)
	if err != nil {
		return "", err
	}
	return time.Since(now).String(), nil
}

func InsertMongo(client *mongo.Client, projects []Project) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	for _, project := range projects {
		_, err = client.Database("test").Collection("projects").InsertOne(ctx, bson.M{
			"name": project.Name,
			"identifier": project.Identifier,
			"invite_code": project.InviteCode,
			"sprint_duration": project.SprintDuration,
			"owner": bson.M{
				"username": project.Owner.Username,
				"first_name": project.Owner.FirstName,
				"last_name": project.Owner.LastName,
			},
			"sprints": bson.A{
				bson.M{
					"name": project.Sprints[0].Name,
					"start_date": project.Sprints[0].StartDate,
					"end_date": project.Sprints[0].EndDate,
				},
			},
		})
		if err != nil {
			return "", err
		}
	}
	return time.Since(now).String(), nil
}

func FindMongo(client *mongo.Client) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection("projects").Find(ctx, bson.M{})
	if err != nil {
		return "", err
	}
	cursor.Close(ctx)

	return time.Since(now).String(), nil
}

func FindMongoWithFilter(client *mongo.Client) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection("projects").Find(ctx, bson.M{"sprint_duration": bson.M{"$gt": 50}})
	if err != nil {
		return "", err
	}
	cursor.Close(ctx)

	return time.Since(now).String(), nil
}

func FindMongoWithFilterAndProjection(client *mongo.Client) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection("projects").Find(
		ctx,
		bson.M{
			"sprint_duration": bson.M{"$gt": 50},
		},
		options.Find().SetProjection(bson.M{
			"name": 1,
			"identifier": 1,
			"invite_code": 1,
			"owner": bson.M{
				"username": 1,
				"first_name": 1,
				"last_name": 1,
			},
			"sprints": bson.M{
				"name": 1,
				"start_date": 1,
				"end_date": 1,
			},
		}),
	)
	if err != nil {
		return "", err
	}
	cursor.Close(ctx)

	return time.Since(now).String(), nil
}

func FindMongoWithFilterAndProjectionAndSort(client *mongo.Client) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection("projects").Find(
		ctx,
		bson.M{
			"sprint_duration": bson.M{"$gt": 50},
		},
		options.Find().SetProjection(bson.M{
			"name": 1,
			"identifier": 1,
			"invite_code": 1,
			"owner": bson.M{
				"username": 1,
				"first_name": 1,
				"last_name": 1,
			},
			"sprints": bson.M{
				"name": 1,
				"start_date": 1,
				"end_date": 1,
			},
		}).SetSort(bson.M{
			"sprints.start_date": -1,
		}),
	)
	if err != nil {
		return "", err
	}
	cursor.Close(ctx)

	return time.Since(now).String(), nil
}

func UpdateMongo(client *mongo.Client) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	_, err = client.Database("test").Collection("projects").UpdateMany(ctx, bson.M{}, bson.M{"$inc": bson.M{"sprint_duration": (60*60*24)}})
	if err != nil {
		return "", err
	}

	return time.Since(now).String(), nil
}

func DeleteMongo(client *mongo.Client) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	_, err = client.Database("test").Collection("projects").DeleteMany(
		ctx,
		bson.M{},
	)
	if err != nil {
		return "", err
	}

	return time.Since(now).String(), nil
}

/* TABLE */

func PrintTable(rows [][]string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"#", "Query", "Postgres", "Mongo"})
	for _, row := range rows {
		t.AppendRow(table.Row{1000, row[0], row[1], row[2]})
	}
	t.Render()
}