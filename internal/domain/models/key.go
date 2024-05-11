package models

type Key struct {
	Label string  `bson:"label"`
	User  KeyUser `bson:"user"`
}

type KeyUser struct {
	ID    string `bson:"id"`
	Name  string `bson:"name"`
	Email string `bson:"email"`
}
