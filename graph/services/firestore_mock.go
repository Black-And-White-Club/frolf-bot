//go:build !test
// +build !test

package services

import (
	"context"

	"github.com/romero-jace/tcr-bot/graph/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockFirestoreClient struct {
	users map[string]*model.User
}

func (m *MockFirestoreClient) Collection(name string) CollectionRef {
	return &MockCollectionRef{m: m}
}

type MockCollectionRef struct {
	m *MockFirestoreClient
}

func (m *MockCollectionRef) Doc(id string) DocumentRef {
	return &MockDocumentRef{m: m.m, id: id}
}

type MockDocumentRef struct {
	m  *MockFirestoreClient
	id string
}

func (m *MockDocumentRef) Get(ctx context.Context) (*DocumentSnapshot, error) {
	if user, exists := m.m.users[m.id]; exists {
		return &DocumentSnapshot{
			data: map[string]interface{}{
				"DiscordID": user.DiscordID,
				"Name":      user.Name,
			},
		}, nil
	}
	return nil, status.Error(codes.NotFound, "not found")
}

func (m *MockDocumentRef) Set(ctx context.Context, data interface{}) (*WriteResult, error) {
	user := data.(*model.User)
	m.m.users[user.DiscordID] = user
	return &WriteResult{}, nil
}
