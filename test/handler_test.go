// handler.go
package graph

import (
	"net/http"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
)

func TestExampleHandler(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ExampleHandler(tt.args.w, tt.args.r)
		})
	}
}

func Test_respondWithError(t *testing.T) {
	type args struct {
		w       http.ResponseWriter
		code    int
		message string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respondWithError(tt.args.w, tt.args.code, tt.args.message)
		})
	}
}

func Test_respondWithJSON(t *testing.T) {
	type args struct {
		w       http.ResponseWriter
		code    int
		payload Response
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respondWithJSON(tt.args.w, tt.args.code, tt.args.payload)
		})
	}
}

func TestSetupRoutes(t *testing.T) {
	type args struct {
		firestoreClient *firestore.Client
	}
	tests := []struct {
		name string
		args args
		want *chi.Mux
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetupRoutes(tt.args.firestoreClient); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetupRoutes() = %v, want %v", got, tt.want)
			}
		})
	}
}
