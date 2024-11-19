package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Project struct {
	MongoId primitive.ObjectID `bson:"_id"`
	Id int `json:"id"`
	Name string `bson:"name" json:"name"`
	Identifier string `bson:"identifier" json:"identifier"`
	InviteCode string `bson:"invite_code" json:"invite_code"`
	SprintDuration int `bson:"sprint_duration" json:"sprint_duration"`
	Owner User `json:"owner"`
	OwnerId int `json:"owner_id"`
	MongoOwnerId primitive.ObjectID `bson:"owner_id"`
	Sprints []Sprint `json:"sprints"`
}

type User struct {
	MongoId primitive.ObjectID `bson:"_id"`
	Id int `json:"id"`
	Username string `bson:"username" json:"username"`
	FirstName string `bson:"first_name" json:"first_name"`
	LastName string `bson:"last_name" json:"last_name"`
}

type Sprint struct {
	MongoId primitive.ObjectID `bson:"_id"`
	Id int `json:"id"`
	Name string `bson:"name" json:"name"`
	StartDate int64 `bson:"start_date" json:"start_date"`
	EndDate int64 `bson:"end_date" json:"end_date"`
	ProjectId int `json:"project_id"`
	MongoProjectId primitive.ObjectID `bson:"project_id"`
}