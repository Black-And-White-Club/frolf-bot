package services

import (
	"context"

	"cloud.google.com/go/firestore"
)

// MyDocumentRef is a wrapper around firestore.DocumentRef
type MyDocumentRef struct {
	ref *firestore.DocumentRef
}

// Implement the DocumentRef interface
func (m *MyDocumentRef) Get(ctx context.Context) (*DocumentSnapshot, error) {
	docSnapshot, err := m.ref.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &DocumentSnapshot{data: docSnapshot.Data()}, nil
}

func (m *MyDocumentRef) Set(ctx context.Context, data interface{}) (*WriteResult, error) {
	// Call the Firestore Set method
	_, err := m.ref.Set(ctx, data)
	if err != nil {
		return nil, err
	}
	return &WriteResult{}, nil // Return a new WriteResult (or modify as needed)
}

// MyCollectionRef is a wrapper around firestore.CollectionRef
type MyCollectionRef struct {
	ref *firestore.CollectionRef
}

func (m *MyCollectionRef) Doc(id string) DocumentRef {
	return &MyDocumentRef{m.ref.Doc(id)}
}

// FirestoreClient is an interface for Firestore operations
type FirestoreClient interface {
	Collection(name string) CollectionRef
}

// CollectionRef is an interface for collection operations
type CollectionRef interface {
	Doc(id string) DocumentRef
}

// DocumentRef is an interface for document operations
type DocumentRef interface {
	Get(ctx context.Context) (*DocumentSnapshot, error)
	Set(ctx context.Context, data interface{}) (*WriteResult, error)
}

// DocumentSnapshot is a mockable representation of a Firestore document snapshot
type DocumentSnapshot struct {
	data map[string]interface{}
}

// DataTo maps the document data to the provided struct
func (ds *DocumentSnapshot) DataTo(v interface{}) error {
	// Implement the mapping logic here
	// For simplicity, we can use reflection or a simple type assertion
	return nil
}

// WriteResult is a mockable representation of a Firestore write result
type WriteResult struct{}

// FirestoreClientWrapper wraps the Firestore client
type FirestoreClientWrapper struct {
	client *firestore.Client
}

func (fw *FirestoreClientWrapper) Collection(name string) CollectionRef {
	return &MyCollectionRef{fw.client.Collection(name)}
}
