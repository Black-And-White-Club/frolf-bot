package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPublisher mocks the message publisher for testing
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(subject string, messages ...*message.Message) error {
	args := m.Called(subject, messages)
	return args.Error(0)
}

func (m *MockPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPublisher) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	args := m.Called(ctx, topic)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan *message.Message), args.Error(1)
}

// MockLoggerAdapter mocks the logger adapter for testing
type MockLoggerAdapter struct {
	mock.Mock
}

func (m *MockLoggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	m.Called(msg, err, fields)
}

func (m *MockLoggerAdapter) Debug(msg string, fields watermill.LogFields) {
	m.Called(msg, fields)
}

func (m *MockLoggerAdapter) Info(msg string, fields watermill.LogFields) {
	m.Called(msg, fields)
}

func (m *MockLoggerAdapter) Trace(msg string, fields watermill.LogFields) {
	m.Called(msg, fields)
}

func (m *MockLoggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	args := m.Called(fields)
	return args.Get(0).(watermill.LoggerAdapter)
}

type MockMessage struct {
	mock.Mock
}

func (m *MockMessage) UUID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMessage) Payload() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *MockMessage) Metadata() message.Metadata { // Correct return type
	args := m.Called()
	if args.Get(0) == nil {
		return message.Metadata{}
	}
	return args.Get(0).(message.Metadata)
}

func (m *MockMessage) Ack() {
	m.Called()
}

func (m *MockMessage) Nack() {
	m.Called()
}

func TestNewMessageHandlers(t *testing.T) {
	type args struct {
		publisher message.Publisher
		logger    watermill.LoggerAdapter
	}
	tests := []struct {
		name string
		args args
		want *MessageHandlers
	}{
		{
			name: "Create new message handlers",
			args: args{
				publisher: nil,
				logger:    nil,
			},
			want: &MessageHandlers{
				Publisher: nil,
				logger:    nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMessageHandlers(tt.args.publisher, tt.args.logger)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMessageHandlers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageHandlers_HandleMessage_UserSignupRequest(t *testing.T) {
	// Create mock objects
	mockPublisher := new(MockPublisher)
	mockLogger := new(MockLoggerAdapter)

	// Create a sample incoming message
	signupReq := userevents.UserSignupRequest{
		DiscordID: "test-discord-id",
		TagNumber: 1,
	}
	msgPayload, err := json.Marshal(signupReq)
	if err != nil {
		t.Fatal(err)
	}
	incomingMsg := message.NewMessage(watermill.NewUUID(), msgPayload)
	incomingMsg.Metadata.Set("subject", userevents.UserSignupRequestSubject)

	// Create the *expected* published message (with a new UUID)
	publishedMsgPayload, _ := json.Marshal(signupReq)
	publishedMsg := message.NewMessage(watermill.NewUUID(), publishedMsgPayload)

	// Define expected behavior of mocks
	mockPublisher.On("Publish", userevents.UserSignupRequestSubject, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		messages := args.Get(1).([]*message.Message) // Correctly get the slice

		if assert.Len(t, messages, 1) { // Assert that only one message was sent
			actualMsg := messages[0]
			assert.Equal(t, publishedMsg.Payload, actualMsg.Payload, "Published message payload should match")
			assert.Equal(t, userevents.UserSignupRequestSubject, args.Get(0).(string), "Published message subject should match")
		}
	}).Once()

	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything).Return()

	// Create message handlers with mocks
	handlers := NewMessageHandlers(mockPublisher, mockLogger)

	// Run the test
	err = handlers.HandleMessage(incomingMsg)

	// Assert on results
	assert.Nil(t, err, "Error occurred while handling message")

	// Verify mock interactions
	mockPublisher.AssertCalled(t, "Publish", userevents.UserSignupRequestSubject, mock.Anything)
	mockLogger.AssertNotCalled(t, "Error", mock.Anything, mock.Anything, mock.Anything)
}

func TestMessageHandlers_HandleMessage_UnknownSubject(t *testing.T) {
	mockPublisher := new(MockPublisher)
	mockLogger := new(MockLoggerAdapter)
	mockMsg := new(MockMessage)
	uuid := watermill.NewUUID()

	// Set up mock expectations
	mockMsg.On("UUID").Return(uuid)
	mockMsg.On("Metadata").Return(map[string]string{"subject": "unknown-subject"})

	logFields := watermill.LogFields{"subject": "unknown-subject", "message_id": uuid}
	mockLogger.On("Error", "Unknown message type", fmt.Errorf("unknown message type: unknown-subject"), logFields).Return()

	handlers := NewMessageHandlers(mockPublisher, mockLogger)

	msg := message.NewMessage(uuid, nil)
	msg.Metadata.Set("subject", "unknown-subject")

	err := handlers.HandleMessage(msg)

	assert.Error(t, err)
	assert.EqualError(t, err, "unknown message type: unknown-subject")
	mockPublisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
	mockLogger.AssertCalled(t, "Error", "Unknown message type", fmt.Errorf("unknown message type: unknown-subject"), logFields)
}

func TestMessageHandlers_HandleMessage_UnmarshalError(t *testing.T) {
	// Create mock objects
	mockPublisher := new(MockPublisher)
	mockLogger := new(MockLoggerAdapter)

	// Create a message with invalid JSON payload
	msg := message.NewMessage(watermill.NewUUID(), []byte("invalid json"))
	msg.Metadata.Set("subject", userevents.UserSignupRequestSubject)

	// Set up mock expectations
	mockLogger.On("Error", "Failed to unmarshal UserSignupRequest", mock.AnythingOfType("*json.SyntaxError"), watermill.LogFields{
		"message_id": msg.UUID,
	}).Return()

	// Create message handlers
	handlers := NewMessageHandlers(mockPublisher, mockLogger)

	// Run the test
	err := handlers.HandleMessage(msg)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal signup request")
	mockPublisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
	mockLogger.AssertCalled(t, "Error", "Failed to unmarshal UserSignupRequest", mock.AnythingOfType("*json.SyntaxError"), watermill.LogFields{
		"message_id": msg.UUID,
	})
}

func TestMessageHandlers_HandleMessage_PublishError(t *testing.T) {
	// Create mock objects
	mockPublisher := new(MockPublisher)
	mockLogger := new(MockLoggerAdapter)

	// Create a valid message
	signupReq := userevents.UserSignupRequest{
		DiscordID: "test-discord-id",
		TagNumber: 1,
	}
	msgPayload, err := json.Marshal(signupReq)
	if err != nil {
		t.Fatal(err)
	}
	incomingMsg := message.NewMessage(watermill.NewUUID(), msgPayload)
	incomingMsg.Metadata.Set("subject", userevents.UserSignupRequestSubject)
	publishedMsgPayload, _ := json.Marshal(signupReq)
	publishedMsg := message.NewMessage(watermill.NewUUID(), publishedMsgPayload)

	// Set up mock expectations: MockPublisher returns an error
	publishErr := errors.New("some publish error")
	mockPublisher.On("Publish", userevents.UserSignupRequestSubject, mock.MatchedBy(func(actual []*message.Message) bool {
		if assert.Len(t, actual, 1) {
			assert.Equal(t, publishedMsg.Payload, actual[0].Payload)
			return true
		}
		return false
	})).Return(publishErr).Once()

	mockLogger.On("Error", "Failed to publish UserSignupRequest", publishErr, watermill.LogFields{
		"message_id": incomingMsg.UUID,
	}).Return()

	// Create message handlers
	handlers := NewMessageHandlers(mockPublisher, mockLogger)

	// Run the test
	err = handlers.HandleMessage(incomingMsg)

	// Assertions
	assert.ErrorIs(t, err, publishErr)
	mockPublisher.AssertCalled(t, "Publish", userevents.UserSignupRequestSubject, mock.Anything)
	mockLogger.AssertCalled(t, "Error", "Failed to publish UserSignupRequest", publishErr, watermill.LogFields{
		"message_id": incomingMsg.UUID,
	})
}

func TestMessageHandlers_HandleMessage_MissingSubject(t *testing.T) {
	mockPublisher := new(MockPublisher)
	mockLogger := new(MockLoggerAdapter)
	mockMsg := new(MockMessage)
	uuid := watermill.NewUUID()

	// Create a REAL message
	msg := message.NewMessage(uuid, nil)

	// Set up mock expectations on the mock
	mockMsg.On("UUID").Return(uuid)
	mockMsg.On("Metadata").Return(map[string]string{}) // Empty metadata

	logFields := watermill.LogFields{"subject": "", "message_id": uuid}
	mockLogger.On("Error", "Unknown message type", fmt.Errorf("unknown message type: "), logFields).Return()

	handlers := NewMessageHandlers(mockPublisher, mockLogger)

	// Call HandleMessage with the REAL message
	err := handlers.HandleMessage(msg)

	assert.Error(t, err)
	assert.EqualError(t, err, "unknown message type: ")
	mockPublisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
	mockLogger.AssertCalled(t, "Error", "Unknown message type", fmt.Errorf("unknown message type: "), logFields)
}

func TestMessageHandlers_handleUserSignupRequest_MarshalError(t *testing.T) {
	mockPublisher := new(MockPublisher)
	mockLogger := new(MockLoggerAdapter)

	uuid := watermill.NewUUID()
	badPayload := []byte("invalid json")

	mockMsg := new(MockMessage)

	// Set up mock expectations
	mockMsg.On("UUID").Return(uuid)
	mockMsg.On("Payload").Return(badPayload)
	mockMsg.On("Metadata").Return(map[string]string{"subject": userevents.UserSignupRequestSubject})
	mockMsg.On("Ack").Return()

	mockLogger.On("Error", "Failed to unmarshal UserSignupRequest", mock.AnythingOfType("*json.SyntaxError"), watermill.LogFields{
		"message_id": uuid,
	}).Return()

	handlers := NewMessageHandlers(mockPublisher, mockLogger)

	// Create a real message and set its metadata
	msg := message.NewMessage(uuid, badPayload)
	msg.Metadata.Set("subject", userevents.UserSignupRequestSubject)

	err := handlers.HandleMessage(msg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal signup request")
	mockPublisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
	mockLogger.AssertCalled(t, "Error", "Failed to unmarshal UserSignupRequest", mock.AnythingOfType("*json.SyntaxError"), watermill.LogFields{
		"message_id": uuid,
	})
}

func TestMessageHandlers_handleUserSignupRequest_AckError(t *testing.T) {
	mockPublisher := new(MockPublisher)
	mockLogger := new(MockLoggerAdapter)
	mockMsg := new(MockMessage)
	uuid := watermill.NewUUID()

	// Create a valid message (for payload comparison)
	signupReq := userevents.UserSignupRequest{
		DiscordID: "test-discord-id",
		TagNumber: 1,
	}
	msgPayload, err := json.Marshal(signupReq)
	if err != nil {
		t.Fatal(err)
	}

	publishedMsgPayload, _ := json.Marshal(signupReq)
	publishedMsg := message.NewMessage(watermill.NewUUID(), publishedMsgPayload)

	// Set up mock expectations
	mockMsg.On("UUID").Return(uuid)
	mockMsg.On("Payload").Return(msgPayload)
	mockMsg.On("Metadata").Return(message.Metadata{"subject": userevents.UserSignupRequestSubject}) // Correct metadata setup
	mockMsg.On("Ack").Return()

	mockPublisher.On("Publish", userevents.UserSignupRequestSubject, mock.MatchedBy(func(actual []*message.Message) bool {
		if assert.Len(t, actual, 1) {
			assert.Equal(t, publishedMsg.Payload, actual[0].Payload)
			return true
		}
		return false
	})).Return(nil).Once()

	mockLogger.On("Info", mock.Anything, mock.Anything).Return()

	handlers := NewMessageHandlers(mockPublisher, mockLogger)

	// Create a real message and set its metadata
	msg := message.NewMessage(uuid, msgPayload)
	msg.Metadata.Set("subject", userevents.UserSignupRequestSubject)

	err = handlers.HandleMessage(msg)

	assert.NoError(t, err)
	mockPublisher.AssertCalled(t, "Publish", userevents.UserSignupRequestSubject, mock.Anything)
	mockLogger.AssertNotCalled(t, "Error", mock.Anything, mock.Anything, mock.Anything)
	// mockMsg.AssertCalled(t, "Ack")
}
