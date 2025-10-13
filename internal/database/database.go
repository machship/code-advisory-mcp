package database

import (
	"context"
	"fmt"

	"github.com/philippgille/chromem-go"
)

// Database wraps the chromem-go vector database.
type Database struct {
	db         *chromem.DB
	collection *chromem.Collection
}

// Document represents a document to be stored in the database.
type Document struct {
	ID        string
	Content   string
	Embedding []float32
	Metadata  map[string]string
}

// New creates a new Database instance or loads from existing path.
func New(path string) (*Database, error) {
	db, err := chromem.NewPersistentDB(path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Try to get existing collection or create new one
	collection := db.GetCollection("advisories", nil)
	if collection == nil {
		// Collection doesn't exist, create it
		collection, err = db.CreateCollection("advisories", nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	return &Database{
		db:         db,
		collection: collection,
	}, nil
}

// Exists checks if a document with the given ID already exists.
func (d *Database) Exists(ctx context.Context, id string) (bool, error) {
	// Try to get the document by ID
	_, err := d.collection.GetByID(ctx, id)
	if err != nil {
		// Document doesn't exist or error occurred
		// chromem-go returns an error if not found
		return false, nil
	}

	return true, nil
}

// Add adds a document to the database.
func (d *Database) Add(ctx context.Context, doc Document) error {
	err := d.collection.AddDocument(ctx, chromem.Document{
		ID:        doc.ID,
		Content:   doc.Content,
		Embedding: doc.Embedding,
		Metadata:  doc.Metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to add document: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	// chromem-go persists automatically, no explicit close needed
	return nil
}
