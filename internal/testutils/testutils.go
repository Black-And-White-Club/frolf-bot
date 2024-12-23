package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

// MockPublisher is the gomock mock for the message.Publisher interface.
type MockPublisher struct {
	ctrl     *gomock.Controller
	recorder *MockPublisherMockRecorder
}

// MockPublisherMockRecorder records expectations for MockPublisher.
type MockPublisherMockRecorder struct {
	mock *MockPublisher
}

// NewMockPublisher creates a new MockPublisher.
func NewMockPublisher(ctrl *gomock.Controller) *MockPublisher {
	mock := &MockPublisher{ctrl: ctrl}
	mock.recorder = &MockPublisherMockRecorder{mock: mock}
	return mock
}

// EXPECT returns the recorder for MockPublisher.
func (m *MockPublisher) EXPECT() *MockPublisherMockRecorder {
	return m.recorder
}

// Publish mocks the Publish method of the Publisher interface.
func (m *MockPublisher) Publish(topic string, messages ...*message.Message) error {
	m.ctrl.T.Helper()
	fmt.Printf("MockPublisher.Publish called with topic: %s\n", topic) // Debugging print statement
	args := []interface{}{topic}
	for _, msg := range messages {
		fmt.Printf("MockPublisher.Publish received messages: %v\n", messages) // Print the received messages
		args = append(args, msg)
	}
	results := m.ctrl.Call(m, "Publish", args...)
	if results[0] != nil {
		return results[0].(error)
	}
	return nil
}

// Publish sets an expectation for the Publish method.
func (mr *MockPublisherMockRecorder) Publish(topic interface{}, messages ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	args := append([]interface{}{topic}, messages...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Publish", reflect.TypeOf((*MockPublisher)(nil).Publish), args...)
}

// Close mocks the Close method of the Publisher interface.
func (m *MockPublisher) Close() error {
	m.ctrl.T.Helper()
	results := m.ctrl.Call(m, "Close")
	if results[0] != nil {
		return results[0].(error)
	}
	return nil
}

// Close sets an expectation for the Close method.
func (mr *MockPublisherMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockPublisher)(nil).Close))
}

type MockSubscriberWithReturn struct {
	ReturnChannel     <-chan *message.Message
	ReturnError       error
	hasBeenSubscribed bool
}

func (m *MockSubscriberWithReturn) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	fmt.Println("MockSubscriberWithReturn.Subscribe called") // Add this print statement
	if m.hasBeenSubscribed {
		fmt.Println("MockSubscriberWithReturn.Subscribe called AGAIN. This is likely an error")
	}
	m.hasBeenSubscribed = true
	return m.ReturnChannel, m.ReturnError
}

func (m *MockSubscriberWithReturn) Messages() <-chan *message.Message {
	return m.ReturnChannel // Implement Messages()
}

func (m *MockSubscriberWithReturn) Close() error {
	return nil
}

// MockLogger is the gomock mock for the watermill.LoggerAdapter interface.
type MockLogger struct {
	ctrl     *gomock.Controller
	recorder *MockLoggerMockRecorder
}

// MockLoggerMockRecorder records expectations for MockLogger.
type MockLoggerMockRecorder struct {
	mock *MockLogger
}

// NewMockLogger creates a new MockLogger.
func NewMockLogger(ctrl *gomock.Controller) *MockLogger {
	mock := &MockLogger{ctrl: ctrl}
	mock.recorder = &MockLoggerMockRecorder{mock: mock}
	return mock
}

// EXPECT returns the recorder for MockLogger.
func (m *MockLogger) EXPECT() *MockLoggerMockRecorder {
	return m.recorder
}

// Error mocks the Error method of the LoggerAdapter interface.
func (m *MockLogger) Error(msg string, err error, fields watermill.LogFields) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Error", msg, err, fields)
}

// Error sets an expectation for the Error method.
func (mr *MockLoggerMockRecorder) Error(msg, err, fields interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock, "Error", reflect.TypeOf((*MockLogger)(nil).Error), msg, err, fields)
}

// Debug mocks the Debug method of the LoggerAdapter interface.
func (m *MockLogger) Debug(msg string, fields watermill.LogFields) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Debug", msg, fields)
}

// Debug sets an expectation for the Debug method.
func (mr *MockLoggerMockRecorder) Debug(msg, fields interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock, "Debug", reflect.TypeOf((*MockLogger)(nil).Debug), msg, fields)
}

// Info mocks the Info method of the LoggerAdapter interface.
func (m *MockLogger) Info(msg string, fields watermill.LogFields) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Info", msg, fields)
}

// Info sets an expectation for the Info method.
func (mr *MockLoggerMockRecorder) Info(msg, fields interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock, "Info", reflect.TypeOf((*MockLogger)(nil).Info), msg, fields)
}

// Trace mocks the Trace method of the LoggerAdapter interface.
func (m *MockLogger) Trace(msg string, fields watermill.LogFields) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Trace", msg, fields)
}

// Trace sets an expectation for the Trace method.
func (mr *MockLoggerMockRecorder) Trace(msg, fields interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock, "Trace", reflect.TypeOf((*MockLogger)(nil).Trace), msg, fields)
}

// With mocks the With method of the LoggerAdapter interface.
func (m *MockLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	m.ctrl.T.Helper()
	results := m.ctrl.Call(m, "With", fields)
	return results[0].(watermill.LoggerAdapter)
}

// With sets an expectation for the With method.
func (mr *MockLoggerMockRecorder) With(fields interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock, "With", reflect.TypeOf((*MockLogger)(nil).With), fields)
}

// start
// MockSubscriber is the gomock mock for the message.Subscriber interface.
type MockSubscriber struct {
	ctrl     *gomock.Controller
	recorder *MockSubscriberMockRecorder
}

// MockSubscriberMockRecorder records expectations for MockSubscriber.
type MockSubscriberMockRecorder struct {
	mock *MockSubscriber
}

// NewMockSubscriber creates a new MockSubscriber.
func NewMockSubscriber(ctrl *gomock.Controller) *MockSubscriber {
	mock := &MockSubscriber{ctrl: ctrl}
	mock.recorder = &MockSubscriberMockRecorder{mock: mock}
	return mock
}

// EXPECT returns the recorder for MockSubscriber.
func (m *MockSubscriber) EXPECT() *MockSubscriberMockRecorder {
	return m.recorder
}

// Subscribe mocks the Subscribe method of the Subscriber interface.
func (m *MockSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subscribe", ctx, topic)
	r0, _ := ret[0].(<-chan *message.Message)
	r1, _ := ret[1].(error)
	return r0, r1
}

// Subscribe sets an expectation for the Subscribe method.
func (mr *MockSubscriberMockRecorder) Subscribe(ctx, topic interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockSubscriber)(nil).Subscribe), ctx, topic)
}

// Messages mocks the Messages method of the Subscriber interface.
func (m *MockSubscriber) Messages() <-chan *message.Message {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Messages")
	r0, _ := ret[0].(<-chan *message.Message)
	return r0
}

// Messages sets an expectation for the Messages method.
func (mr *MockSubscriberMockRecorder) Messages() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Messages", reflect.TypeOf((*MockSubscriber)(nil).Messages))
}

// Close mocks the Close method of the Subscriber interface.
func (m *MockSubscriber) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	r0, _ := ret[0].(error)
	return r0
}

// Close sets an expectation for the Close method.
func (mr *MockSubscriberMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockSubscriber)(nil).Close))
}

// end
// MockSubscribeResult is a helper struct to return from mock Subscribe calls
type MockSubscribeResult struct {
	Channel <-chan *message.Message
	Error   error
}

// CreateMessageWithPayload creates a Watermill message with a JSON payload.
func CreateMessageWithPayload(t *testing.T, correlationID string, payload interface{}) *message.Message {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("correlation_id", correlationID)
	return msg
}

// NewMessageMatcher creates a gomock matcher that checks if the message payload
// matches the expected payload after marshaling to JSON.
func NewMessageMatcher(t *testing.T, expected interface{}) gomock.Matcher {
	return &messageMatcher{t: t, expected: expected}
}

type messageMatcher struct {
	t        *testing.T
	expected interface{}
}

func (m *messageMatcher) Matches(x interface{}) bool {
	msg, ok := x.(*message.Message)
	if !ok {
		m.t.Errorf("Expected *message.Message, got %T", x)
		return false
	}

	expectedJSON, err := json.Marshal(m.expected)
	if err != nil {
		m.t.Errorf("Failed to marshal expected payload: %v", err)
		return false
	}

	return reflect.DeepEqual(msg.Payload, expectedJSON)
}

func (m *messageMatcher) String() string {
	return fmt.Sprintf("matches message with payload %+v", m.expected)
}
