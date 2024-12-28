package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/mock"
)

// Mock RoundDB
type mockRoundDB struct {
	mock.Mock
	rounddb.RoundDB // Embed the interface
}

func (m *mockRoundDB) GetParticipantsWithResponses(ctx context.Context, roundID string, responses ...rounddb.Response) ([]rounddb.Participant, error) {
	args := m.Called(ctx, roundID, responses)
	return args.Get(0).([]rounddb.Participant), args.Error(1)
}

func (m *mockRoundDB) CreateRound(ctx context.Context, round *rounddb.Round) error {
	args := m.Called(ctx, round)
	return args.Error(0)
}

func (m *mockRoundDB) GetRound(ctx context.Context, roundID string) (*rounddb.Round, error) {
	args := m.Called(ctx, roundID)
	return args.Get(0).(*rounddb.Round), args.Error(1)
}

func (m *mockRoundDB) GetRoundState(ctx context.Context, roundID string) (rounddb.RoundState, error) {
	args := m.Called(ctx, roundID)
	return args.Get(0).(rounddb.RoundState), args.Error(1)
}

func (m *mockRoundDB) UpdateRound(ctx context.Context, roundID string, round *rounddb.Round) error {
	args := m.Called(ctx, roundID, round)
	return args.Error(0)
}

func (m *mockRoundDB) DeleteRound(ctx context.Context, roundID string) error {
	args := m.Called(ctx, roundID)
	return args.Error(0)
}

func (m *mockRoundDB) LogRound(ctx context.Context, round *rounddb.Round, updateType rounddb.ScoreUpdateType) error {
	args := m.Called(ctx, round, updateType)
	return args.Error(0)
}

func (m *mockRoundDB) UpdateParticipant(ctx context.Context, roundID string, participant rounddb.Participant) error {
	args := m.Called(ctx, roundID, participant)
	return args.Error(0)
}

func (m *mockRoundDB) UpdateRoundState(ctx context.Context, roundID string, state rounddb.RoundState) error {
	args := m.Called(ctx, roundID, state)
	return args.Error(0)
}

func (m *mockRoundDB) GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*rounddb.Round, error) {
	args := m.Called(ctx, now, oneHourFromNow)
	return args.Get(0).([]*rounddb.Round), args.Error(1)
}

func (m *mockRoundDB) UpdateParticipantScore(ctx context.Context, roundID, participantID string, score int) error {
	args := m.Called(ctx, roundID, participantID, score)
	return args.Error(0)
}

// Mock Publisher
type mockPublisher struct {
	mock.Mock
}

func (m *mockPublisher) Publish(topic string, messages ...*message.Message) error {
	args := m.Called(topic, messages)
	return args.Error(0)
}

func (m *mockPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Mock Subscriber
type mockSubscriber struct {
	mock.Mock
}

func (m *mockSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	args := m.Called(ctx, topic)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan *message.Message), args.Error(1)
}

func (m *mockSubscriber) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Mock LoggerAdapter
type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Info(msg string, fields watermill.LogFields) {
	m.Called(msg, fields)
}

func (m *mockLogger) Debug(msg string, fields watermill.LogFields) {
	m.Called(msg, fields)
}

func (m *mockLogger) Trace(msg string, fields watermill.LogFields) {
	m.Called(msg, fields)
}

func (m *mockLogger) Error(msg string, err error, fields watermill.LogFields) {
	m.Called(msg, err, fields)
}

func (m *mockLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	args := m.Called(fields)
	return args.Get(0).(watermill.LoggerAdapter)
}

// Test file
func TestRoundService_getTagNumber(t *testing.T) {
	ctx := context.Background()
	mockPublisher := &mockPublisher{}
	mockSubscriber := &mockSubscriber{}
	mockLogger := &mockLogger{}
	mockRoundDB := &mockRoundDB{} // Still needed for the RoundService struct

	s := &RoundService{
		RoundDB:    mockRoundDB,
		Publisher:  mockPublisher,
		Subscriber: mockSubscriber,
		logger:     mockLogger,
	}

	correlationID := watermill.NewUUID()
	validResponse := &roundevents.GetTagNumberResponseEvent{TagNumber: 42}
	validResponseData, _ := json.Marshal(validResponse)

	tests := []struct {
		name        string
		setupMocks  func()
		discordID   string
		expectedTag *int
		expectedErr bool
	}{
		{
			name: "Successfully retrieve tag number",
			setupMocks: func() {
				mockPublisher.On("Publish", roundevents.GetTagNumberRequestSubject, mock.Anything).
					Return(nil).Once()

				responseChan := make(chan *message.Message, 1)
				responseChan <- message.NewMessage(correlationID, validResponseData)
				close(responseChan)

				mockSubscriber.On("Subscribe", ctx, roundevents.GetTagNumberResponseSubject).
					Return((<-chan *message.Message)(responseChan), nil).Once()
			},
			discordID:   "123456789",
			expectedTag: &validResponse.TagNumber,
			expectedErr: false,
		},
		{
			name: "Failure to publish request",
			setupMocks: func() {
				mockPublisher.On("Publish", roundevents.GetTagNumberRequestSubject, mock.Anything).
					Return(fmt.Errorf("publish error")).Once()
			},
			discordID:   "123456789",
			expectedTag: nil,
			expectedErr: true,
		},
		{
			name: "Failure to subscribe to response",
			setupMocks: func() {
				mockPublisher.On("Publish", roundevents.GetTagNumberRequestSubject, mock.Anything).
					Return(nil).Once()
				mockSubscriber.On("Subscribe", ctx, roundevents.GetTagNumberResponseSubject).
					Return(nil, fmt.Errorf("subscribe error")).Once()
			},
			discordID:   "123456789",
			expectedTag: nil,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			tag, err := s.getTagNumber(ctx, tt.discordID)

			if (err != nil) != tt.expectedErr {
				t.Errorf("getTagNumber() error = %v, expectedErr = %v", err, tt.expectedErr)
			}

			if tt.expectedTag != nil { // Check if expectedTag is not nil
				if tag == nil { // Check if tag is nil
					t.Errorf("getTagNumber() = nil, expected = %v", *tt.expectedTag)
				} else if *tag != *tt.expectedTag { // Now safe to dereference
					t.Errorf("getTagNumber() = %v, expected = %v", *tag, *tt.expectedTag)
				}
			} else if tag != nil { // Handle the case where expectedTag is nil but tag is not
				t.Errorf("getTagNumber() = %v, expected = nil", *tag)
			}

			mockPublisher.AssertExpectations(t)
			mockSubscriber.AssertExpectations(t)
			mockRoundDB.AssertExpectations(t)
		})
	}
}

func TestRoundService_getUserRole(t *testing.T) {
	ctx := context.Background()
	mockPublisher := &mockPublisher{}
	mockSubscriber := &mockSubscriber{}
	mockLogger := &mockLogger{}
	mockRoundDB := &mockRoundDB{}

	s := &RoundService{
		RoundDB:    mockRoundDB,
		Publisher:  mockPublisher,
		Subscriber: mockSubscriber,
		logger:     mockLogger,
	}

	correlationID := watermill.NewUUID()
	validResponse := &roundevents.GetUserRoleResponseEvent{Role: "admin"}
	validResponseData, _ := json.Marshal(validResponse)

	tests := []struct {
		name         string
		setupMocks   func()
		discordID    string
		expectedRole string
		expectedErr  bool
	}{
		{
			name: "Successfully retrieve user role",
			setupMocks: func() { // Pass correlationID to setupMocks
				mockPublisher.On("Publish", roundevents.GetUserRoleRequestSubject, mock.Anything).
					Return(nil).Once()

				responseChan := make(chan *message.Message, 1)
				responseChan <- message.NewMessage(correlationID, validResponseData) // Use the passed correlationID
				close(responseChan)

				mockSubscriber.On("Subscribe", ctx, roundevents.GetUserRoleResponseSubject).
					Return((<-chan *message.Message)(responseChan), nil).Once()
			},
			discordID:    "123456789",
			expectedRole: "admin",
			expectedErr:  false,
		},
		{
			name: "Failure to subscribe to response",
			setupMocks: func() {
				mockPublisher.On("Publish", roundevents.GetUserRoleRequestSubject, mock.Anything).
					Return(nil).Once()
				mockSubscriber.On("Subscribe", ctx, roundevents.GetUserRoleResponseSubject).
					Return(nil, fmt.Errorf("subscribe error")).Once()
			},
			discordID:    "123456789",
			expectedRole: "",
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			role, err := s.getUserRole(ctx, tt.discordID)

			if (err != nil) != tt.expectedErr {
				t.Errorf("getUserRole() error = %v, expectedErr = %v", err, tt.expectedErr)
			}
			if role != tt.expectedRole {
				t.Errorf("getUserRole() = %v, expected = %v", role, tt.expectedRole)
			}
		})
	}
}
