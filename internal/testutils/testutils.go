package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	types "github.com/Black-And-White-Club/tcr-bot/app/types"
	"go.uber.org/mock/gomock"
)

type MockSubscriberWithReturn struct {
	ReturnChannel     <-chan types.Message // Use types.Message
	ReturnError       error
	hasBeenSubscribed bool
}

func (m *MockSubscriberWithReturn) Subscribe(ctx context.Context, topic string) (<-chan types.Message, error) {
	fmt.Println("MockSubscriberWithReturn.Subscribe called")
	if m.hasBeenSubscribed {
		fmt.Println("MockSubscriberWithReturn.Subscribe called AGAIN. This is likely an error")
	}
	m.hasBeenSubscribed = true
	return m.ReturnChannel, m.ReturnError
}

func (m *MockSubscriberWithReturn) Messages() <-chan types.Message {
	return m.ReturnChannel
}

func (m *MockSubscriberWithReturn) Close() error {
	return nil
}

// CreateMessageWithPayload creates a MockMessage with a JSON payload.
func CreateMessageWithPayload(t *testing.T, correlationID string, payload interface{}) types.Message {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	ctrl := gomock.NewController(t) // Create a gomock controller

	// Create a new MockMessage using the generated mock
	msg := NewMockMessage(ctrl)
	msg.EXPECT().UUID().Return(types.NewUUID())
	msg.EXPECT().Payload().Return(payloadBytes)
	msg.EXPECT().Metadata().Return(map[string]string{
		"correlation_id": correlationID,
	})

	return msg
}

// NewMessageMatcher is a matcher for Watermill messages
func NewMessageMatcher(t *testing.T, expectedPayload interface{}) gomock.Matcher {
	return &MessageMatcher{
		t:               t,
		expectedPayload: expectedPayload,
	}
}

// MessageMatcher is a gomock.Matcher that matches a message.Message based on its payload.
type MessageMatcher struct {
	t               *testing.T
	expectedPayload interface{}
}

func (m *MessageMatcher) Matches(x interface{}) bool {
	msg, ok := x.(types.Message)
	if !ok {
		return false
	}

	// Unmarshal the msg.Payload into a temporary map
	var actualPayload map[string]interface{}
	err := json.Unmarshal(msg.Payload(), &actualPayload)
	if err != nil {
		m.t.Errorf("Failed to unmarshal actual payload: %v", err)
		return false
	}

	// Convert expectedPayload to map[string]interface{} for comparison
	expectedPayloadMap, err := convertToMap(m.expectedPayload)
	if err != nil {
		m.t.Errorf("Failed to convert expected payload to map: %v", err)
		return false
	}

	// Special handling for UserRole types (compare using String() method)
	if _, ok := actualPayload["NewRole"].(string); ok {
		if expectedRole, ok := expectedPayloadMap["NewRole"].(map[string]interface{}); ok {
			if roleInterface, ok := expectedRole["String"]; ok {
				if roleFunc, ok := roleInterface.(func() string); ok {
					expectedPayloadMap["NewRole"] = roleFunc()
				}
			}
		}
	}

	fmt.Printf("Actual Payload: %+v\n", actualPayload)
	fmt.Printf("Expected Payload: %+v\n", expectedPayloadMap)

	// Compare the two maps using reflect.DeepEqual
	return reflect.DeepEqual(actualPayload, expectedPayloadMap)
}

func (m *MessageMatcher) String() string {
	return fmt.Sprintf("matches message with payload %v", m.expectedPayload)
}

// Helper function to convert expected payload to map[string]interface{}
func convertToMap(payload interface{}) (map[string]interface{}, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var payloadMap map[string]interface{}
	err = json.Unmarshal(payloadJSON, &payloadMap)
	if err != nil {
		return nil, err
	}

	return payloadMap, nil
}

// Custom matcher to check if an error is of a specific type
type isErrorTypeMatcher struct {
	expectedType reflect.Type
}

func (m isErrorTypeMatcher) Matches(x interface{}) bool {
	err, ok := x.(error)
	if !ok {
		return false
	}
	return reflect.TypeOf(err) == m.expectedType
}

func (m isErrorTypeMatcher) String() string {
	return fmt.Sprintf("is of type %v", m.expectedType)
}

// Helper function to create a new isErrorTypeMatcher
func IsErrorType(expectedType interface{}) gomock.Matcher {
	return isErrorTypeMatcher{reflect.TypeOf(expectedType)}
}

// ContainsStringMatcher is a custom matcher that checks if a string contains a specific substring.
type ContainsStringMatcher struct {
	Expected string
}

// Matches returns whether x is a string that contains the expected substring.
func (m ContainsStringMatcher) Matches(x interface{}) bool {
	s, ok := x.(string)
	if !ok {
		return false
	}
	return strings.Contains(s, m.Expected)
}

// String returns a description of the matcher.
func (m ContainsStringMatcher) String() string {
	return fmt.Sprintf("contains the substring %q", m.Expected)
}

// ContainsString is a helper function to create a ContainsStringMatcher.
func ContainsString(expected string) gomock.Matcher {
	return &ContainsStringMatcher{Expected: expected}
}
