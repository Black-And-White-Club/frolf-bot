package service

import (
	"context"

	"cloud.google.com/go/firestore"
)

var FirestoreClient *firestore.Client

func InitializeFirestore(ctx context.Context) error {
	var err error
	FirestoreClient, err = firestore.NewClient(ctx, "your-project-id")
	if err != nil {
		return err
	}
	return nil
}
