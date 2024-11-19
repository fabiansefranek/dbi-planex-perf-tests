package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/jackc/pgx/v5"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/fabiansefranek/dbi-perf-tests/db"
	"github.com/fabiansefranek/dbi-perf-tests/models"
)

const (
	atlasConnectionString = "mongodb+srv://dbi:dbi@cluster0.rmuwo.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"
)

func main() {
	/* --- START CONTAINERS --- */
	println("Starting Postgres container...")
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

	db.PostgresConn = postgresConn

	println("Initializing Postgres...")
	err = InitializePostgres(postgresConn)
	if err != nil {
		panic(err)
	}

	println("Starting MongoDB container...")
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

	mongoConn, err := ConnectMongoDB(mongoConnectionString, false)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := mongoConn.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()

	db.MongoConn = mongoConn

	err = InitializeMongoDB(mongoConn)
	if err != nil {
		panic(err)
	}

	StartServer()

	return

	mongoAtlasConn, err := ConnectMongoDB(atlasConnectionString, true)
	if err != nil {
		panic(err)
	}

	/* --- SCHEMA VALIDATION TEST --- */

	err = InsertMongoSchemaViolation(mongoConn)
	if err != nil {
		panic(err)
	}

	/* --- PERFORMANCE TESTS --- */

	sizes := []int{100, 1000/*, 10000*/} // TODO: Recompile charts!
	tableRows := make([][]string, 0)

	postgresInsertData, mongoInsertData := make([]opts.LineData, 0), make([]opts.LineData, 0)
	postgresFindData, mongoFindData := make([]opts.LineData, 0), make([]opts.LineData, 0)
	postgresUpdateData, mongoUpdateData := make([]opts.LineData, 0), make([]opts.LineData, 0)
	postgresDeleteData, mongoDeleteData := make([]opts.LineData, 0), make([]opts.LineData, 0)

	println("Starting performance tests...")
	for _, size := range sizes {
		sizeAsString := fmt.Sprint(size)
		projects := GenerateProjects(size)

		/* INSERT */

		postgresDurationNumeric, err := InsertPostgres(postgresConn, projects)
		if err != nil {
			panic(err)
		}

		mongoDurationNumeric, err := InsertMongo(mongoConn, projects, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationWithIndexNumeric, err := InsertMongo(mongoConn, projects, "projects_index")
		if err != nil {
			panic(err)
		}

		mongoDurationAtlasNumeric, err := InsertMongo(mongoAtlasConn, projects, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationRefercing, err := InsertMongoWithReferencing(mongoConn, projects)
		if err != nil {
			panic(err)
		}

		tableRows = append(tableRows, []string{sizeAsString, "Insert", postgresDurationNumeric.String(), mongoDurationNumeric.String(), mongoDurationWithIndexNumeric.String(), mongoDurationAtlasNumeric.String(), mongoDurationRefercing})

		postgresInsertData = append(postgresInsertData, opts.LineData{Value: postgresDurationNumeric.Milliseconds()})
		mongoInsertData = append(mongoInsertData, opts.LineData{Value: mongoDurationNumeric.Milliseconds()})

		/* FIND */

		postgresDuration, err := FindPostgres(postgresConn)
		if err != nil {
			panic(err)
		}

		mongoDuration, err := FindMongo(mongoConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationWithIndex, err := FindMongo(mongoConn, "projects_index")
		if err != nil {
			panic(err)
		}

		mongoDurationAtlas, err := FindMongo(mongoAtlasConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationReferencing, err := FindMongoWithReferencing(mongoConn)
		if err != nil {
			panic(err)
		}

		tableRows = append(tableRows, []string{sizeAsString, "Find", postgresDuration, mongoDuration, mongoDurationWithIndex, mongoDurationAtlas, mongoDurationReferencing})

		postgresDurationNumeric, err = FindPostgresWithFilter(postgresConn)
		if err != nil {
			panic(err)
		}

		mongoDurationNumeric, err = FindMongoWithFilter(mongoConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationWithIndexNumeric, err = FindMongoWithFilter(mongoConn, "projects_index")
		if err != nil {
			panic(err)
		}

		mongoDurationAtlasNumeric, err = FindMongoWithFilter(mongoAtlasConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationWithReferencing, err := FindMongoWithReferencing(mongoConn)
		if err != nil {
			panic(err)
		}

		tableRows = append(tableRows, []string{sizeAsString, "Find with filter", postgresDurationNumeric.String(), mongoDurationNumeric.String(), mongoDurationWithIndexNumeric.String(), mongoDurationAtlasNumeric.String(), mongoDurationWithReferencing})

		postgresFindData = append(postgresFindData, opts.LineData{Value: postgresDurationNumeric.Milliseconds()})
		mongoFindData = append(mongoFindData, opts.LineData{Value: mongoDurationNumeric.Milliseconds()})

		postgresDuration, err = FindPostgresWithFilterAndProjection(postgresConn)
		if err != nil {
			panic(err)
		}

		mongoDuration, err = FindMongoWithFilterAndProjection(mongoConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationWithIndex, err = FindMongoWithFilterAndProjection(mongoConn, "projects_index")
		if err != nil {
			panic(err)
		}

		mongoDurationAtlas, err = FindMongoWithFilterAndProjection(mongoAtlasConn, "projects")
		if err != nil {
			panic(err)
		}

		tableRows = append(tableRows, []string{sizeAsString, "Find with filter and projection", postgresDuration, mongoDuration, mongoDurationWithIndex, mongoDurationAtlas, "-"})

		postgresDuration, err = FindPostgresWithFilterAndProjectionAndSort(postgresConn)
		if err != nil {
			panic(err)
		}

		mongoDuration, err = FindMongoWithFilterAndProjectionAndSort(mongoConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationWithIndex, err = FindMongoWithFilterAndProjectionAndSort(mongoConn, "projects_index")
		if err != nil {
			panic(err)
		}

		mongoDurationAtlas, err = FindMongoWithFilterAndProjectionAndSort(mongoAtlasConn, "projects")
		if err != nil {
			panic(err)
		}

		tableRows = append(tableRows, []string{sizeAsString, "Find with filter and projection and sort", postgresDuration, mongoDuration, mongoDurationWithIndex, mongoDurationAtlas, "-"})

		postgresDuration, err = FindPostgresWithAggregation(postgresConn)
		if err != nil {
			panic(err)
		}

		mongoDuration, err = FindMongoWithAggregation(mongoConn, "projects")
		if err != nil {
			panic(err)
		}

		tableRows = append(tableRows, []string{sizeAsString, "Find with aggregation", postgresDuration, mongoDuration, "-", "-", "-"})

		/* UPDATE */

		postgresDurationNumeric, err = UpdatePostgres(postgresConn)
		if err != nil {
			panic(err)
		}

		mongoDurationNumeric, err = UpdateMongo(mongoConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationWithIndexNumeric, err = UpdateMongo(mongoConn, "projects_index")
		if err != nil {
			panic(err)
		}

		mongoDurationAtlasNumeric, err = UpdateMongo(mongoAtlasConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationReferencing, err = UpdateMongoWithReferencing(mongoConn)
		if err != nil {
			panic(err)
		}

		tableRows = append(tableRows, []string{sizeAsString, "Update", postgresDurationNumeric.String(), mongoDurationNumeric.String(), mongoDurationWithIndexNumeric.String(), mongoDurationAtlasNumeric.String(), mongoDurationReferencing})

		postgresUpdateData = append(postgresUpdateData, opts.LineData{Value: postgresDurationNumeric.Milliseconds()})
		mongoUpdateData = append(mongoUpdateData, opts.LineData{Value: mongoDurationNumeric.Milliseconds()})

		/* DELETE */

		postgresDurationNumeric, err = DeletePostgres(postgresConn)
		if err != nil {
			panic(err)
		}

		mongoDurationNumeric, err = DeleteMongo(mongoConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationWithIndexNumeric, err = DeleteMongo(mongoConn, "projects_index")
		if err != nil {
			panic(err)
		}

		mongoDurationAtlasNumeric, err = DeleteMongo(mongoAtlasConn, "projects")
		if err != nil {
			panic(err)
		}

		mongoDurationReferencing, err = DeleteMongoWithReferencing(mongoConn)
		if err != nil {
			panic(err)
		}
		
		tableRows = append(tableRows, []string{sizeAsString, "Delete", postgresDurationNumeric.String(), mongoDurationNumeric.String(), mongoDurationWithIndexNumeric.String(), mongoDurationAtlasNumeric.String(), mongoDurationReferencing})

		postgresDeleteData = append(postgresDeleteData, opts.LineData{Value: postgresDurationNumeric.Milliseconds()})
		mongoDeleteData = append(mongoDeleteData, opts.LineData{Value: mongoDurationNumeric.Milliseconds()})

		tableRows = append(tableRows, []string{})

		println("Finished test size ", size)
	}

	PrintTable(tableRows)

	/* --- CHARTS --- */
	insertLineChart := CreateLineChart("Insert", sizes, postgresInsertData, mongoInsertData)
	findLineChart := CreateLineChart("Find (With Filter)", sizes, postgresFindData, mongoFindData)
	updateLineChart := CreateLineChart("Update", sizes, postgresUpdateData, mongoUpdateData)
	deleteLineChart := CreateLineChart("Delete", sizes, postgresDeleteData, mongoDeleteData)

	page := components.NewPage()
	page.AddCharts(insertLineChart, findLineChart, updateLineChart, deleteLineChart)
	f, err := os.Create("charts.html")
	if err != nil {
		panic(err)
	}
	page.Render(io.MultiWriter(f))

	println("Performance tests finished")

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
	mongodbContainer, err := mongodb.Run(ctx, "mongo:latest", mongodb.WithUsername("user"), mongodb.WithPassword("password"))

	if err != nil {
		return "", nil, err
	}

	connectionString, err = mongodbContainer.ConnectionString(ctx)
	if err != nil {
		return "", nil, err
	}

	return connectionString, mongodbContainer, nil
}

func ConnectMongoDB(connectionString string, atlas bool) (client *mongo.Client, err error) {
	opts := options.Client().ApplyURI(connectionString)
	if (atlas) {
		opts = opts.SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1))
	}
	client, err = mongo.Connect(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func InitializeMongoDB(client *mongo.Client) (err error) {
	validator := bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"name", "identifier", "invite_code", "sprint_duration", "owner", "sprints"},
			"properties": bson.M{
				"id": bson.M{
					"bsonType": "int",
				},
				"name": bson.M{
					"bsonType":    "string",
					"description": "The project name",
				},
				"identifier": bson.M{
					"bsonType":    "string",
					"description": "Unique project identifier",
				},
				"invite_code": bson.M{
					"bsonType":    "string",
					"description": "Code used to invite others to the project",
				},
				"sprint_duration": bson.M{
					"bsonType":    "int",
					"description": "The duration of a sprint",
				},
				"owner": bson.M{
					"bsonType": "object",
					"items": bson.M{
						"bsonType": "object",
						"required": []string{"username", "first_name", "last_name"},
						"properties": bson.M{
							"username": bson.M{
								"bsonType": "string",
							},
							"first_name": bson.M{
								"bsonType": "string",
							},
							"last_name": bson.M{
								"bsonType": "string",
							},
						},
					},
				},
				"sprints": bson.M{
					"bsonType": "array",
					"items": bson.M{
						"bsonType": "object",
						"required": []string{"name", "start_date", "end_date"},
						"properties": bson.M{
							"name": bson.M{
								"bsonType": "string",
							},
							"start_date": bson.M{
								"bsonType": "long", 
							},
							"end_date": bson.M{
								"bsonType": "long",
							},
						},
					},
				},
			},
		},
	}

	
	err = client.Database("test").CreateCollection(context.Background(), "projects_index", &options.CreateCollectionOptions{
		Validator: &validator,
	})
	if err != nil {
		return err
	}

	coll := client.Database("test").Collection("projects_index")

	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "sprint_duration", Value: 1}},
	}
	_, err = coll.Indexes().CreateOne(context.Background(), indexModel)
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

func GenerateProjects(arraySize int) []models.Project {
	var result []models.Project
	for i := 0; i < arraySize; i++ {
		result = append(result, models.Project{
			Name: RandomString(10),
			Identifier: RandomString(48),
			InviteCode: RandomString(128),
			SprintDuration: rand.Intn(100) + 1,
			Owner: models.User{
				Username: RandomString(10),
				FirstName: RandomString(10),
				LastName: RandomString(10),
			},
			Sprints: []models.Sprint{
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

func InsertPostgres(conn *pgx.Conn, projects []models.Project) (duration time.Duration, err error) {
	now := time.Now()
	for _, project := range projects {
		var userId int
		err = conn.QueryRow(context.Background(), 
			`INSERT INTO users (username, first_name, last_name) VALUES ($1, $2, $3) RETURNING id;`,
			project.Owner.Username, project.Owner.FirstName, project.Owner.LastName).Scan(&userId)
		if err != nil {
			return time.Since(now), err
		}

		var projectId int
		err = conn.QueryRow(context.Background(),
			`INSERT INTO projects (name, identifier, invite_code, sprint_duration, owner_id) VALUES ($1, $2, $3, $4, $5) RETURNING id;`,
			project.Name, project.Identifier, project.InviteCode, project.SprintDuration, userId).Scan(&projectId)
		if err != nil {
			return time.Since(now), err
		}

		for _, sprint := range project.Sprints {
			var sprintId int
			err = conn.QueryRow(context.Background(),
				`INSERT INTO sprints (name, project_id, start_date, end_date) VALUES ($1, $2, $3, $4) RETURNING id;`,
				sprint.Name, projectId, sprint.StartDate, sprint.EndDate).Scan(&sprintId)
			if err != nil {
				return time.Since(now), err
			}
		}
	}

	return time.Since(now), nil
}

func FindPostgres(conn *pgx.Conn) (duration string, err error) {
	now := time.Now()	
	_, err = conn.Exec(context.Background(), `SELECT * FROM sprints INNER JOIN projects ON sprints.project_id = projects.id INNER JOIN users ON projects.owner_id = users.id;`)
	if err != nil {
		return "", err
	}

	return time.Since(now).String(), nil
}

func FindPostgresWithFilter(conn *pgx.Conn) (duration time.Duration, err error) {
	now := time.Now()	
	_, err = conn.Exec(context.Background(), `SELECT * FROM sprints INNER JOIN projects ON sprints.project_id = projects.id INNER JOIN users ON projects.owner_id = users.id WHERE projects.sprint_duration > 50;`)
	if err != nil {
		return time.Since(now), err
	}

	return time.Since(now), nil
}

func FindPostgresWithFilterAndProjection(conn *pgx.Conn) (duration string, err error) {
	now := time.Now()
	_, err = conn.Exec(context.Background(), `SELECT users.username, projects.name, sprints.name FROM sprints INNER JOIN projects ON sprints.project_id = projects.id INNER JOIN users ON projects.owner_id = users.id WHERE projects.sprint_duration > 50;`)
	if err != nil {
		return "", err
	}

	return time.Since(now).String(), nil
}

func FindPostgresWithAggregation(conn *pgx.Conn) (duration string, err error) {
	now := time.Now()
	_, err = conn.Exec(context.Background(), `SELECT users.username AS owner, COUNT(*) AS count FROM projects INNER JOIN users ON projects.owner_id = users.id GROUP BY owner;`)
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

func UpdatePostgres(conn *pgx.Conn) (duration time.Duration, err error) {
	now := time.Now()
	_, err = conn.Exec(context.Background(), `UPDATE sprints SET start_date = start_date + (60*60*24)`)
	if err != nil {
		return time.Since(now), err
	}
	return time.Since(now), nil
}

func DeletePostgres(conn *pgx.Conn) (duration time.Duration, err error) {
	now := time.Now()
	_, err = conn.Exec(context.Background(), `DELETE FROM sprints`)
	if err != nil {
		return time.Since(now), err
	}
	return time.Since(now), nil
}

func InsertMongo(client *mongo.Client, projects []models.Project, collection string) (duration time.Duration, err error) {
	now := time.Now()
	ctx := context.Background()
	for _, project := range projects {
		_, err = client.Database("test").Collection(collection).InsertOne(ctx, bson.M{
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
			return time.Since(now), err
		}
	}
	return time.Since(now), nil
}

func InsertMongoSchemaViolation(client *mongo.Client) (err error) {
	_, err = client.Database("test").Collection("projects_index").InsertOne(context.Background(), bson.M{
		"identifier": "PX",
		"invite_code": "AS)D(Zaihz2e)",
		"owner": bson.M{
			"username": "maxmuster",
			"first_name": "Max",
			"last_name": "Muster",
		},
		"sprints": bson.A{
			bson.M{
				"name": "Sprint 1",
				"start_date": 0,
				"end_date": 1,
			},
		},
	})
	if err == nil {
		return errors.New("insert is not violating schema, but should")
	}

	println("Test Insert successfully violated schema")
	return nil
}

func InsertMongoWithReferencing(client *mongo.Client, projects []models.Project) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	for _, project := range projects {
		result, err := client.Database("test").Collection("users").InsertOne(ctx, bson.M{
			"username": project.Owner.Username,
			"first_name": project.Owner.FirstName,
			"last_name": project.Owner.LastName,
		})
		if err != nil {
			return "", err
		}

		ownerId := result.InsertedID


		result, err = client.Database("test").Collection("projects").InsertOne(ctx, bson.M{
			"name": project.Name,
			"identifier": project.Identifier,
			"invite_code": project.InviteCode,
			"sprint_duration": project.SprintDuration,
			"owner": ownerId,
		})
		if err != nil {
			return "", err
		}

		projectId := result.InsertedID

		for _, sprint := range project.Sprints {
			_, err = client.Database("test").Collection("sprints").InsertOne(ctx, bson.M{
				"name": sprint.Name,
				"project": projectId,
				"start_date": sprint.StartDate,
				"end_date": sprint.EndDate,
			})
			if err != nil {
				return "", err
			}
		}
	}
	return time.Since(now).String(), nil
}

func FindMongo(client *mongo.Client, collection string) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection(collection).Find(ctx, bson.M{})
	if err != nil {
		return "", err
	}
	cursor.Close(ctx)

	return time.Since(now).String(), nil
}

func FindMongoWithFilter(client *mongo.Client, collection string) (duration time.Duration, err error) {
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection(collection).Find(ctx, bson.M{"sprint_duration": bson.M{"$gt": 50}})
	if err != nil {
		return time.Since(now), err
	}
	cursor.Close(ctx)

	return time.Since(now), nil
}

func FindMongoWithFilterAndProjection(client *mongo.Client, collection string) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection(collection).Find(
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

func FindMongoWithFilterAndProjectionAndSort(client *mongo.Client, collection string) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection(collection).Find(
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

func FindMongoWithAggregation(client *mongo.Client, collection string) (duration string, err error) {
	// Count projects per owner 
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection(collection).Aggregate(ctx, []bson.M{
		{
			"$unwind": "$owner",
		},
		{
			"$group": bson.M{
				"_id": "$owner.username",
				"count": bson.M{"$sum": 1},
			},
		},
	})
	if err != nil {
		return "", err
	}

	cursor.Close(ctx)
	return time.Since(now).String(), nil
}

func FindMongoWithReferencing(client *mongo.Client) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	cursor, err := client.Database("test").Collection("projects").Find(ctx, bson.M{})
	if err != nil {
		return "", err
	}
	cursor.Close(ctx)

	cursor, err = client.Database("test").Collection("users").Find(ctx, bson.M{})
	if err != nil {
		return "", err
	}
	cursor.Close(ctx)

	cursor, err = client.Database("test").Collection("sprints").Find(ctx, bson.M{})
	if err != nil {
		return "", err
	}
	cursor.Close(ctx)

	return time.Since(now).String(), nil	
}

func UpdateMongo(client *mongo.Client, collection string) (duration time.Duration, err error) {
	now := time.Now()
	ctx := context.Background()
	_, err = client.Database("test").Collection(collection).UpdateMany(ctx, bson.M{}, bson.M{"$inc": bson.M{"sprint_duration": (60*60*24)}})
	if err != nil {
		return time.Since(now), err
	}

	return time.Since(now), nil
}

func UpdateMongoWithReferencing(client *mongo.Client) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	_, err = client.Database("test").Collection("projects").UpdateMany(ctx, bson.M{}, bson.M{"$inc": bson.M{"sprint_duration": (60*60*24)}})
	if err != nil {
		return "", err
	}

	return time.Since(now).String(), nil
}

func DeleteMongo(client *mongo.Client, collection string) (duration time.Duration, err error) {
	now := time.Now()
	ctx := context.Background()
	_, err = client.Database("test").Collection(collection).DeleteMany(
		ctx,
		bson.M{},
	)
	if err != nil {
		return time.Since(now), err
	}

	return time.Since(now), nil
}

func DeleteMongoWithReferencing(client *mongo.Client) (duration string, err error) {
	now := time.Now()
	ctx := context.Background()
	_, err = client.Database("test").Collection("projects").DeleteMany(
		ctx,
		bson.M{},
	)
	if err != nil {
		return "", err
	}

	_, err = client.Database("test").Collection("users").DeleteMany(
		ctx,
		bson.M{},
	)
	if err != nil {
		return "", err
	}

	_, err = client.Database("test").Collection("sprints").DeleteMany(
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
	t.AppendHeader(table.Row{"#", "Query", "Postgres", "Mongo", "Mongo (Index)", "Mongo (Atlas)", "Mongo (Referencing)"})
	for _, row := range rows {
		if len(row) != 7 {
			t.AppendSeparator()
			continue
		}
		t.AppendRow(table.Row{row[0], row[1], row[2], row[3], row[4], row[5], row[6]})
	}
	t.Render()
}

/* CHARTS */

func CreateLineChart(title string, sizes []int, postgresData []opts.LineData, mongoData []opts.LineData) *charts.Line {
	lineChart := charts.NewLine()

	lineChart.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: title}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Batch Size"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Time (ms)"}),
	)

	lineChart.SetXAxis(sizes).AddSeries("Postgres", postgresData).AddSeries("Mongo", mongoData)

	return lineChart
}