package mongodb

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"tms/internal/domain/models"
	"tms/internal/storage"
)

type Storage struct {
	client   *mongo.Client
	database string
}

// New creates a new instance of the MongoDB storage.
func New(uri string, database string) (*Storage, error) {
	const op = "storage.mongodb.New"

	clientOptions := options.Client().ApplyURI(uri)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		client:   client,
		database: database,
	}, nil
}

// Close closes instated mongodb connection.
func (s *Storage) Close(ctx context.Context) error {
	if s.client != nil {
		return s.client.Disconnect(ctx)
	}
	return nil
}

func (s *Storage) UpdateUser(ctx context.Context, email string, updateData bson.M) error {
	const op = "storage.mongodb.UpdateUser"

	collection := s.client.Database(s.database).Collection("users")
	filter := bson.M{"email": email}

	update := bson.M{"$set": updateData}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrorUserNotFound)
	}

	return nil
}

func (s *Storage) DeleteUser(ctx context.Context, email string) error {
	const op = "storage.mongodb.DeleteUser"

	collection := s.client.Database(s.database).Collection("users")
	filter := bson.M{
		"email": email,
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrorUserNotFound)
	}

	return nil
}

func (s *Storage) SaveKeyPair(
	ctx context.Context,
	keyLabel string,
	userId string,
) error {
	const op = "storage.mongodb.SaveKeyPair"

	filter := bson.M{"uniqueId": userId}
	var user models.User

	collection := s.client.Database(s.database).Collection("users")
	err := collection.FindOne(ctx, filter).Decode(&user)

	if err != nil {
		return fmt.Errorf("%s: %s", op, "User not found")
	}

	collection = s.client.Database(s.database).Collection("keyPairs")
	doc := bson.M{
		"label": keyLabel,
		"user": bson.M{
			"id":    userId,
			"name":  user.Name,
			"email": user.Email,
		},
	}

	_, err = collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) DeleteKeyPair(
	ctx context.Context,
	userId string,
	keyLabel string,
) (bool, error) {
	const op = "storage.mongodb.DeleteKeys"

	collection := s.client.Database(s.database).Collection("keyPairs")

	result, err := collection.DeleteOne(
		ctx, bson.M{
			"label": keyLabel,
			"user": bson.M{
				"id": userId,
			},
		})

	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if result.DeletedCount == 0 {
		return false, fmt.Errorf("%s: No matched key pair was deleted", op)
	}

	return true, nil
}

func (s *Storage) SaveDocument(
	ctx context.Context,
	title string,
	ownerId string,
	content string,
) (string, error) {
	const op = "storage.mongodb.SaveDocument"

	filter := bson.M{"uniqueId": ownerId}
	var user models.User

	collection := s.client.Database(s.database).Collection("users")
	err := collection.FindOne(ctx, filter).Decode(&user)

	if err != nil {
		return "", fmt.Errorf("%s: User not found", op)
	}

	collection = s.client.Database(s.database).Collection("documents")

	document := bson.D{
		{Key: "title", Value: title},
		{Key: "content", Value: content},
		{Key: "owner", Value: bson.D{
			{Key: "id", Value: ownerId},
			{Key: "name", Value: user.Name},
			{Key: "email", Value: user.Email},
		}},
	}

	result, err := collection.InsertOne(ctx, document)
	if err != nil {

	}
	if _, ok := result.InsertedID.(primitive.ObjectID); ok {
		return result.InsertedID.(primitive.ObjectID).Hex(), nil
	}

	return "", fmt.Errorf("%s: failed to get inserted document ID", op)
}

func (s *Storage) GetDocument(ctx context.Context, id string) (models.Document, error) {
	const op = "storage.mongodb.Document"

	collection := s.client.Database(s.database).Collection("documents")

	documentId, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": documentId}

	var document models.Document

	err := collection.FindOne(ctx, filter).Decode(&document)

	if err != nil {
		return models.Document{}, fmt.Errorf("%s: %w", op, err)
	}

	return document, nil
}

func (s *Storage) UpdateDocument(
	ctx context.Context,
	id string,
	title string,
	content string,
	ownerId string,
) (models.Document, error) {
	const op = "storage.mongodb.UpdateDocument"

	documentId, _ := primitive.ObjectIDFromHex(id)

	var owner models.User

	filter := bson.M{"uniqueId": ownerId}

	collection := s.client.Database(s.database).Collection("users")
	err := collection.FindOne(ctx, filter).Decode(&owner)

	if err != nil {
		return models.Document{}, fmt.Errorf("%s: %w", op, "Owner not found")
	}

	collection = s.client.Database(s.database).Collection("documents")

	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "title", Value: title},
			{Key: "content", Value: content},
			{Key: "owner", Value: bson.D{
				{Key: "id", Value: ownerId},
				{Key: "name", Value: owner.Name},
				{Key: "email", Value: owner.Email},
			}},
		}},
	}

	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": documentId},
		update,
	)

	if err != nil {
		return models.Document{}, fmt.Errorf("%s: %w", op, err)
	}

	return models.Document{
		Id:      id,
		Title:   title,
		Content: content,
		Owner: models.Owner{
			Id:    ownerId,
			Name:  owner.Name,
			Email: owner.Email,
		},
	}, nil
}

func (s *Storage) DeleteDocument(ctx context.Context, id string) (bool, error) {
	const op = "storage.mongodb.DeleteDocument"

	documentId, _ := primitive.ObjectIDFromHex(id)
	collection := s.client.Database(s.database).Collection("documents")

	result, err := collection.DeleteOne(ctx, bson.M{"_id": documentId})

	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if result.DeletedCount == 0 {
		return false, fmt.Errorf("%s: No matched document was deleted", op)
	}

	return true, nil
}

func (s *Storage) SaveUser(
	ctx context.Context,
	id string,
	name string,
	email string,
) (models.User, error) {
	collection := s.client.Database(s.database).Collection("users")
	user := models.User{UniqueId: id, Name: name, Email: email}

	opts := options.Replace().SetUpsert(true)
	filter := bson.M{"uniqueId": id}
	_, err := collection.ReplaceOne(ctx, filter, user, opts)
	if err != nil {
		return models.User{}, fmt.Errorf("could not save user: %w", err)
	}
	return user, nil
}

func (s *Storage) RemoveUser(
	ctx context.Context,
	id string,
) (bool, error) {
	collection := s.client.Database(s.database).Collection("users")
	filter := bson.M{"_id": id}
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("could not remove user: %w", err)
	}
	return result.DeletedCount > 0, nil
}

func (s *Storage) GetUser(
	ctx context.Context,
	id string,
) (models.User, error) {
	collection := s.client.Database(s.database).Collection("users")
	filter := bson.M{"_id": id}
	var user models.User
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.User{}, fmt.Errorf("no user found with id %s", id)
		}
		return models.User{}, fmt.Errorf("could not retrieve user: %w", err)
	}
	return user, nil
}
