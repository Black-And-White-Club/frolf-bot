package database

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

var FirestoreClient *firestore.Client

func InitializeFirestore(ctx context.Context) error {
	var err error

	// Check if we are running in the emulator
	if os.Getenv("FIRESTORE_EMULATOR_HOST") != "" {
		// Connect to the Firestore emulator
		FirestoreClient, err = firestore.NewClient(ctx, "your-project-id", option.WithoutAuthentication())
		if err != nil {
			return err
		}
		log.Println("Connected to Firestore emulator")
	} else {
		// Connect to the production Firestore
		FirestoreClient, err = firestore.NewClient(ctx, "your-project-id")
		if err != nil {
			return err
		}
		log.Println("Connected to Firestore production")
	}

	return nil
}
