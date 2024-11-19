package db

import (
	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	PostgresConn *pgx.Conn
	MongoConn    *mongo.Client
)