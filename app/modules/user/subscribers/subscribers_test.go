package usersubscribers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	user_mocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserSubscribers_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSubscriber := testutils.NewMockSubscriber(ctrl)

	s := &UserSubscribers{
		subscriber: mockSubscriber,
		handlers:   nil, // Not relevant for this test
		logger:     watermill.NopLogger{},
	}

	t.Run("Success", func(t *testing.T) {
		mockSubscriber.EXPECT().Close().Return(nil)

		if err := s.Close(); err != nil {
			t.Errorf("UserSubscribers.Close() error = %v", err)
		}
	})

	t.Run("Error", func(t *testing.T) {
		mockSubscriber.EXPECT().Close().Return(errors.New("subscriber close error"))

		if err := s.Close(); err == nil {
			t.Error("UserSubscribers.Close() expected an error, but got nil")
		}
	})
}

func TestNewUserSubscribers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSubscriber := testutils.NewMockSubscriber(ctrl)
	mockHandlers := user_mocks.NewMockHandlers(ctrl)

	t.Run("Success", func(t *testing.T) {
		got, got1, err := NewUserSubscribers(mockSubscriber, mockHandlers, watermill.NopLogger{})
		if err != nil {
			t.Errorf("NewUserSubscribers() error = %v", err)
			return
		}
		if got == nil {
			t.Error("NewUserSubscribers() got = nil, want not nil")
		}
		if got1 == nil {
			t.Error("NewUserSubscribers() got1 = nil, want not nil")
		}
	})

	t.Run("Nil Subscriber", func(t *testing.T) {
		_, _, err := NewUserSubscribers(nil, mockHandlers, watermill.NopLogger{})
		if err == nil {
			t.Error("NewUserSubscribers() expected an error for nil subscriber, but got nil")
		}
	})

	t.Run("Nil Handlers", func(t *testing.T) {
		_, _, err := NewUserSubscribers(mockSubscriber, nil, watermill.NopLogger{})
		if err == nil {
			t.Error("NewUserSubscribers() expected an error for nil handlers, but got nil")
		}
	})

	t.Run("Nil Logger", func(t *testing.T) {
		_, _, err := NewUserSubscribers(mockSubscriber, mockHandlers, nil)
		if err == nil {
			t.Error("NewUserSubscribers() expected an error for nil logger, but got nil")
		}
	})
}

func TestUserSubscribers_SubscribeToUserEvents(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*testutils.MockSubscriber, *user_mocks.MockHandlers)
		wantErr    bool
	}{
		{
			name: "Success",
			setupMocks: func(mockSubscriber *testutils.MockSubscriber, mockHandlers *user_mocks.MockHandlers) {
				ctx := context.Background()
				// Expect subscriptions to the subjects
				mockSubscriber.EXPECT().Subscribe(ctx, userevents.UserSignupRequestSubject).Return(make(<-chan *message.Message), nil)
				mockSubscriber.EXPECT().Subscribe(ctx, userevents.UserRoleUpdateRequestSubject).Return(make(<-chan *message.Message), nil)
			},
			wantErr: false,
		},
		{
			name: "Subscribe Error",
			setupMocks: func(mockSubscriber *testutils.MockSubscriber, mockHandlers *user_mocks.MockHandlers) {
				ctx := context.Background()
				// Expect an error on the first subscription
				mockSubscriber.EXPECT().Subscribe(ctx, userevents.UserSignupRequestSubject).Return(nil, errors.New("subscribe error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSubscriber := testutils.NewMockSubscriber(ctrl)
			mockHandlers := user_mocks.NewMockHandlers(ctrl)
			s := &UserSubscribers{
				subscriber: mockSubscriber,
				handlers:   mockHandlers,
				logger:     watermill.NopLogger{},
			}
			tt.setupMocks(mockSubscriber, mockHandlers)
			ctx := context.Background()
			if err := s.SubscribeToUserEvents(ctx); (err != nil) != tt.wantErr {
				t.Errorf("UserSubscribers.SubscribeToUserEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_processEventMessages(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(chan *message.Message, context.Context, context.CancelFunc)
	}{
		{
			name: "Context Cancelled",
			setupMocks: func(messages chan *message.Message, ctx context.Context, cancel context.CancelFunc) {
				var wg sync.WaitGroup
				wg.Add(1)

				go func() {
					defer wg.Done()
					processEventMessages(ctx, messages, func(context.Context, *message.Message) error { return nil }, watermill.NopLogger{})
				}()

				cancel() // Cancel the context immediately
				wg.Wait()
			},
		},
		{
			name: "Messages Channel Closed",
			setupMocks: func(messages chan *message.Message, ctx context.Context, cancel context.CancelFunc) {
				var wg sync.WaitGroup
				wg.Add(1)

				go func() {
					defer wg.Done()
					processEventMessages(ctx, messages, func(context.Context, *message.Message) error { return nil }, watermill.NopLogger{})
				}()

				// Close the channel after a delay
				time.AfterFunc(50*time.Millisecond, func() {
					close(messages)
				})

				wg.Wait()
			},
		},

		/* TODO FIX AT SOME POINT */
		// {
		// 	name: "Handler Error",
		// 	setupMocks: func(messages chan *message.Message, ctx context.Context, cancel context.CancelFunc) {
		// 		mockMsg := message.NewMessage(watermill.NewUUID(), []byte("test payload"))
		// 		messages <- mockMsg

		// 		handlerWithError := func(ctx context.Context, msg *message.Message) error {
		// 			return errors.New("handler error")
		// 		}

		// 		var wg sync.WaitGroup
		// 		wg.Add(1)
		// 		go func() {
		// 			defer wg.Done()
		// 			processEventMessages(ctx, messages, handlerWithError, watermill.NopLogger{})
		// 		}()

		// 		time.Sleep(100 * time.Millisecond) // Allow time for message processing
		// 		cancel()
		// 		wg.Wait()

		// 		// Check if the message was negatively acknowledged (Nack)
		// 		select {
		// 		case <-mockMsg.Nacked():
		// 			// Success, message was Nacked
		// 		case <-time.After(100 * time.Millisecond):
		// 			t.Errorf("processEventMessages() did not Nack the message")
		// 		}
		// 	},
		// },
		// {
		// 	name: "Success",
		// 	setupMocks: func(messages chan *message.Message, ctx context.Context, cancel context.CancelFunc) {
		// 		mockMsg := message.NewMessage(watermill.NewUUID(), []byte("test payload"))
		// 		messages <- mockMsg

		// 		handler := func(ctx context.Context, msg *message.Message) error {
		// 			return nil
		// 		}

		// 		var wg sync.WaitGroup
		// 		wg.Add(1)
		// 		go func() {
		// 			defer wg.Done()
		// 			processEventMessages(ctx, messages, handler, watermill.NopLogger{})
		// 		}()

		// 		// Send the "STOP" message to terminate gracefully
		// 		stopMsg := message.NewMessage(watermill.NewUUID(), []byte("STOP"))
		// 		messages <- stopMsg

		// 		wg.Wait()

		// 		// Check if the message was acknowledged (Ack)
		// 		select {
		// 		case <-mockMsg.Acked():
		// 			// Success, message was Acked
		// 		case <-time.After(100 * time.Millisecond):
		// 			t.Errorf("processEventMessages() did not Ack the message")
		// 		}
		// 	},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			messages := make(chan *message.Message)

			tt.setupMocks(messages, ctx, cancel)
		})
	}
}
