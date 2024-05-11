package models

type User struct {
	UniqueId string `bson:"uniqueId"`
	Name     string `bson:"name"`
	Email    string `bson:"email"`
}
