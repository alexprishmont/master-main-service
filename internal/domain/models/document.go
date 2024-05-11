package models

type Document struct {
	Id      string `bson:"_id"`
	Title   string `bson:"title"`
	Content string `bson:"content"`
	Owner   Owner  `bson:"owner"`
}

type Owner struct {
	Id    string `bson:"_id"`
	Name  string `bson:"name"`
	Email string `bson:"email"`
}
