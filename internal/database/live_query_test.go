package database

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/nfrund/goby/internal/domain"
	"github.com/stretchr/testify/suite"
	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

type LiveQueryTestSuite struct {
	suite.Suite
	conn    *Connection
	service *SurrealLiveQueryService
	cleanup func()
}

func (suite *LiveQueryTestSuite) SetupSuite() {
	if testing.Short() {
		suite.T().Skip("skipping integration test in short mode")
	}

	conn, _, cleanup := setupTestDB(suite.T())
	suite.conn = conn
	suite.service = NewSurrealLiveQueryService(conn)
	suite.cleanup = cleanup
}

func (suite *LiveQueryTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func TestLiveQueryService(t *testing.T) {
	suite.Run(t, new(LiveQueryTestSuite))
}

func (suite *LiveQueryTestSuite) TestSubscribeToTableChanges() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a test user to trigger live query notifications
	client, err := NewClient[TestUser](suite.conn)
	suite.Require().NoError(err)

	// Channel to synchronize notifications with test assertions
	notificationChan := make(chan struct {
		action LiveQueryAction
		data   interface{}
	}, 10)

	handler := func(ctx context.Context, action LiveQueryAction, data interface{}) {
		// Log human-readable version of the data for debugging
		if dataMap, ok := data.(map[string]interface{}); ok {
			if jsonData, err := json.MarshalIndent(dataMap, "", "  "); err == nil {
				slog.Info("Live query notification received", "action", action, "data", string(jsonData))
			}
		}

		// Send notification to test
		select {
		case notificationChan <- struct {
			action LiveQueryAction
			data   interface{}
		}{action, data}:
		case <-ctx.Done():
		}
	}

	// Subscribe to user table changes
	subscription, err := suite.service.Subscribe(ctx, "user", nil, handler)
	suite.Require().NoError(err)
	suite.Require().NotNil(subscription)
	suite.Require().NotEmpty(subscription.ID)
	suite.Equal("user", subscription.Table)
	suite.True(subscription.Active)

	// Give the live query a moment to establish
	time.Sleep(500 * time.Millisecond)

	// Test CREATE action
	testUser := TestUser{
		User: domain.User{
			Name:  stringPtr("Live Query Test User"),
			Email: "livequery@example.com",
		},
		Password: "password",
	}

	createdUser, err := client.Create(ctx, "user", testUser)
	suite.Require().NoError(err)
	defer client.Delete(ctx, createdUser.ID.String())

	// Wait for CREATE notification
	select {
	case notification := <-notificationChan:
		suite.Equal(ActionCreate, notification.action, "Should receive CREATE action")
		// Verify data contains expected fields
		if dataMap, ok := notification.data.(map[string]interface{}); ok {
			suite.Equal("livequery@example.com", dataMap["email"])
		}
	case <-time.After(5 * time.Second):
		suite.Fail("Timeout waiting for CREATE notification")
	}

	// Test UPDATE action
	createdUser.Name = stringPtr("Updated Name")
	_, err = client.Update(ctx, createdUser.ID.String(), createdUser)
	suite.Require().NoError(err)

	// Wait for UPDATE notification
	select {
	case notification := <-notificationChan:
		suite.Equal(ActionUpdate, notification.action, "Should receive UPDATE action")
	case <-time.After(5 * time.Second):
		suite.Fail("Timeout waiting for UPDATE notification")
	}

	// Test DELETE action
	err = client.Delete(ctx, createdUser.ID.String())
	suite.Require().NoError(err)

	// Wait for DELETE notification
	select {
	case notification := <-notificationChan:
		suite.Equal(ActionDelete, notification.action, "Should receive DELETE action")
	case <-time.After(5 * time.Second):
		suite.Fail("Timeout waiting for DELETE notification")
	}

	// Clean up subscription
	err = suite.service.Unsubscribe(subscription.ID)
	suite.NoError(err)
}

func (suite *LiveQueryTestSuite) TestSubscribeQueryWithCustomQuery() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Channel to synchronize notifications
	notificationChan := make(chan struct {
		action LiveQueryAction
		data   interface{}
	}, 10)

	handler := func(ctx context.Context, action LiveQueryAction, data interface{}) {
		// Log human-readable version of the data for debugging
		if dataMap, ok := data.(map[string]interface{}); ok {
			if jsonData, err := json.MarshalIndent(dataMap, "", "  "); err == nil {
				slog.Info("Live query notification received", "action", action, "data", string(jsonData))
			}
		}

		select {
		case notificationChan <- struct {
			action LiveQueryAction
			data   interface{}
		}{action, data}:
		case <-ctx.Done():
		}
	}

	// Test 1: Subscribe with field selection (email and name only)
	query := "LIVE SELECT email, name FROM user WHERE email = $email"
	params := map[string]interface{}{"email": "custom@example.com"}

	subscription, err := suite.service.SubscribeQuery(ctx, query, params, handler)
	suite.Require().NoError(err)
	suite.Require().NotNil(subscription)

	// Give the live query a moment to establish
	time.Sleep(500 * time.Millisecond)

	// Create a user that matches the query filter
	client, err := NewClient[TestUser](suite.conn)
	suite.Require().NoError(err)

	matchingUser := TestUser{
		User: domain.User{
			Name:  stringPtr("Custom Query Test User"),
			Email: "custom@example.com",
		},
		Password: "password",
	}

	createdMatchingUser, err := client.Create(ctx, "user", matchingUser)
	suite.Require().NoError(err)
	defer client.Delete(ctx, createdMatchingUser.ID.String())

	// Wait for notification for matching user
	select {
	case notification := <-notificationChan:
		suite.Equal(ActionCreate, notification.action)
		// Verify field selection - should only have email and name (and id)
		if dataMap, ok := notification.data.(map[string]interface{}); ok {
			suite.Equal("custom@example.com", dataMap["email"], "Should have email field")
			suite.Contains(dataMap, "name", "Should have name field")
			// Note: SurrealDB may include 'id' field automatically
		}
	case <-time.After(5 * time.Second):
		suite.Fail("Timeout waiting for notification for matching user")
	}

	// Test 2: Create a non-matching user - should NOT receive notification
	nonMatchingUser := TestUser{
		User: domain.User{
			Name:  stringPtr("Non-Matching User"),
			Email: "other@example.com",
		},
		Password: "password",
	}

	createdNonMatchingUser, err := client.Create(ctx, "user", nonMatchingUser)
	suite.Require().NoError(err)
	defer client.Delete(ctx, createdNonMatchingUser.ID.String())

	// Wait a bit to ensure no notification arrives
	select {
	case notification := <-notificationChan:
		// If we receive a notification, it should be for the matching user, not the non-matching one
		if dataMap, ok := notification.data.(map[string]interface{}); ok {
			suite.NotEqual("other@example.com", dataMap["email"], "Should not receive notification for non-matching user")
		}
	case <-time.After(2 * time.Second):
		// This is expected - no notification for non-matching user
		slog.Info("Correctly did not receive notification for non-matching user")
	}

	// Clean up subscription
	err = suite.service.Unsubscribe(subscription.ID)
	suite.NoError(err)
}

func (suite *LiveQueryTestSuite) TestMultipleConcurrentSubscriptions() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Channels to track notifications for each table
	userNotifications := make(chan string, 10)
	tableANotifications := make(chan string, 10)
	tableBNotifications := make(chan string, 10)

	// Handler for user table
	userHandler := func(ctx context.Context, action LiveQueryAction, data interface{}) {
		slog.Info("User table notification", "action", action)
		userNotifications <- "user"
	}

	// Handler for table_a
	tableAHandler := func(ctx context.Context, action LiveQueryAction, data interface{}) {
		slog.Info("Table A notification", "action", action)
		tableANotifications <- "table_a"
	}

	// Handler for table_b
	tableBHandler := func(ctx context.Context, action LiveQueryAction, data interface{}) {
		slog.Info("Table B notification", "action", action)
		tableBNotifications <- "table_b"
	}

	// Create 3 subscriptions to different tables
	userSub, err := suite.service.Subscribe(ctx, "user", nil, userHandler)
	suite.Require().NoError(err)
	defer suite.service.Unsubscribe(userSub.ID)

	tableASub, err := suite.service.Subscribe(ctx, "test_table_a", nil, tableAHandler)
	suite.Require().NoError(err)
	defer suite.service.Unsubscribe(tableASub.ID)

	tableBSub, err := suite.service.Subscribe(ctx, "test_table_b", nil, tableBHandler)
	suite.Require().NoError(err)
	defer suite.service.Unsubscribe(tableBSub.ID)

	// Give subscriptions time to establish
	time.Sleep(500 * time.Millisecond)

	// Create clients for different data types
	userClient, err := NewClient[TestUser](suite.conn)
	suite.Require().NoError(err)

	// Use generic client for ad-hoc tables
	type GenericRecord struct {
		ID   *surrealmodels.RecordID `json:"id,omitempty"`
		Name string                  `json:"name"`
	}
	tableAClient, err := NewClient[GenericRecord](suite.conn)
	suite.Require().NoError(err)

	tableBClient, err := NewClient[GenericRecord](suite.conn)
	suite.Require().NoError(err)

	// Trigger changes in all tables concurrently
	var wg sync.WaitGroup
	wg.Add(3)

	// Create user record
	go func() {
		defer wg.Done()
		testUser := TestUser{
			User: domain.User{
				Name:  stringPtr("Concurrent Test User"),
				Email: "concurrent@example.com",
			},
			Password: "password",
		}
		createdUser, err := userClient.Create(ctx, "user", testUser)
		if err == nil {
			defer userClient.Delete(ctx, createdUser.ID.String())
		}
	}()

	// Create table_a record
	go func() {
		defer wg.Done()
		recordA := GenericRecord{Name: "Record A"}
		createdA, err := tableAClient.Create(ctx, "test_table_a", recordA)
		if err == nil {
			defer tableAClient.Delete(ctx, createdA.ID.String())
		}
	}()

	// Create table_b record
	go func() {
		defer wg.Done()
		recordB := GenericRecord{Name: "Record B"}
		createdB, err := tableBClient.Create(ctx, "test_table_b", recordB)
		if err == nil {
			defer tableBClient.Delete(ctx, createdB.ID.String())
		}
	}()

	wg.Wait()

	// Verify each handler receives only its table's notifications
	timeout := time.After(5 * time.Second)
	receivedUser := false
	receivedTableA := false
	receivedTableB := false

	for i := 0; i < 3; i++ {
		select {
		case table := <-userNotifications:
			suite.Equal("user", table)
			receivedUser = true
		case table := <-tableANotifications:
			suite.Equal("table_a", table)
			receivedTableA = true
		case table := <-tableBNotifications:
			suite.Equal("table_b", table)
			receivedTableB = true
		case <-timeout:
			suite.Fail("Timeout waiting for notifications")
			return
		}
	}

	// Verify all subscriptions received their notifications
	suite.True(receivedUser, "User subscription should receive notification")
	suite.True(receivedTableA, "Table A subscription should receive notification")
	suite.True(receivedTableB, "Table B subscription should receive notification")

	// Verify no cross-contamination - channels should be empty now
	select {
	case <-userNotifications:
		suite.Fail("User handler received extra notification")
	case <-tableANotifications:
		suite.Fail("Table A handler received extra notification")
	case <-tableBNotifications:
		suite.Fail("Table B handler received extra notification")
	case <-time.After(1 * time.Second):
		// Expected - no extra notifications
		slog.Info("No cross-contamination detected - test passed")
	}
}

func (suite *LiveQueryTestSuite) TestUnsubscribeRemovesSubscription() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewClient[TestUser](suite.conn)
	suite.Require().NoError(err)

	// Track handler invocations
	handlerInvoked := make(chan bool, 10)
	var invocationCount int
	var mu sync.Mutex

	handler := func(ctx context.Context, action LiveQueryAction, data interface{}) {
		mu.Lock()
		invocationCount++
		mu.Unlock()

		slog.Info("Handler invoked", "action", action, "count", invocationCount)
		handlerInvoked <- true
	}

	// Subscribe
	subscription, err := suite.service.Subscribe(ctx, "user", nil, handler)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(subscription.ID)

	// Give the live query a moment to establish
	time.Sleep(500 * time.Millisecond)

	// Create a user to trigger a notification before unsubscribe
	testUser1 := TestUser{
		User: domain.User{
			Name:  stringPtr("Before Unsubscribe User"),
			Email: "before@example.com",
		},
		Password: "password",
	}

	createdUser1, err := client.Create(ctx, "user", testUser1)
	suite.Require().NoError(err)
	defer client.Delete(ctx, createdUser1.ID.String())

	// Wait for the first notification
	select {
	case <-handlerInvoked:
		slog.Info("Received notification before unsubscribe")
	case <-time.After(5 * time.Second):
		suite.Fail("Should have received notification before unsubscribe")
	}

	// Unsubscribe
	err = suite.service.Unsubscribe(subscription.ID)
	suite.NoError(err)

	// Give time for cleanup (KILL command, etc.)
	time.Sleep(1 * time.Second)

	// Create another user after unsubscribe - should NOT trigger handler
	testUser2 := TestUser{
		User: domain.User{
			Name:  stringPtr("After Unsubscribe User"),
			Email: "after@example.com",
		},
		Password: "password",
	}

	createdUser2, err := client.Create(ctx, "user", testUser2)
	suite.Require().NoError(err)
	defer client.Delete(ctx, createdUser2.ID.String())

	// Wait to ensure no notification arrives
	select {
	case <-handlerInvoked:
		suite.Fail("Handler should not be invoked after unsubscribe")
	case <-time.After(2 * time.Second):
		slog.Info("Correctly did not receive notification after unsubscribe")
	}

	// Verify handler was only invoked once (before unsubscribe)
	mu.Lock()
	finalCount := invocationCount
	mu.Unlock()
	suite.Equal(1, finalCount, "Handler should only be invoked once (before unsubscribe)")

	// Try to unsubscribe again - should not error
	err = suite.service.Unsubscribe(subscription.ID)
	suite.NoError(err, "Double unsubscribe should not error")
}

func (suite *LiveQueryTestSuite) TestSubscribeWithNilHandlerReturnsError() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subscription, err := suite.service.Subscribe(ctx, "user", nil, nil)
	suite.Error(err)
	suite.Nil(subscription)
	suite.Contains(err.Error(), "handler cannot be nil")
}

func (suite *LiveQueryTestSuite) TestSubscribeQueryWithNilHandlerReturnsError() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subscription, err := suite.service.SubscribeQuery(ctx, "LIVE SELECT * FROM user", nil, nil)
	suite.Error(err)
	suite.Nil(subscription)
	suite.Contains(err.Error(), "handler cannot be nil")
}

func (suite *LiveQueryTestSuite) TestHandlerPanic() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewClient[TestUser](suite.conn)
	suite.Require().NoError(err)

	// Channel to track if panic was handled
	panicHandled := make(chan bool, 1)

	// Create a handler that panics
	panicHandler := func(ctx context.Context, action LiveQueryAction, data interface{}) {
		slog.Info("Panic handler invoked - about to panic")
		panicHandled <- true
		panic("intentional test panic")
	}

	// Subscribe with panicking handler
	subscription1, err := suite.service.Subscribe(ctx, "user", nil, panicHandler)
	suite.Require().NoError(err)
	defer suite.service.Unsubscribe(subscription1.ID)

	// Give the live query a moment to establish
	time.Sleep(500 * time.Millisecond)

	// Trigger a notification that will cause the handler to panic
	testUser := TestUser{
		User: domain.User{
			Name:  stringPtr("Panic Test User"),
			Email: "panic@example.com",
		},
		Password: "password",
	}

	createdUser, err := client.Create(ctx, "user", testUser)
	suite.Require().NoError(err)
	defer client.Delete(ctx, createdUser.ID.String())

	// Wait for panic to be handled
	select {
	case <-panicHandled:
		slog.Info("Panic was handled successfully")
	case <-time.After(5 * time.Second):
		suite.Fail("Timeout waiting for panic handler to be invoked")
	}

	// Give the service time to log the panic
	time.Sleep(500 * time.Millisecond)

	// Create a second subscription with a normal handler to verify service still works
	notificationChan := make(chan bool, 1)
	normalHandler := func(ctx context.Context, action LiveQueryAction, data interface{}) {
		slog.Info("Normal handler invoked after panic")
		notificationChan <- true
	}

	subscription2, err := suite.service.Subscribe(ctx, "user", nil, normalHandler)
	suite.Require().NoError(err, "Service should still work after handler panic")
	defer suite.service.Unsubscribe(subscription2.ID)

	// Give the live query a moment to establish
	time.Sleep(500 * time.Millisecond)

	// Trigger another notification
	testUser2 := TestUser{
		User: domain.User{
			Name:  stringPtr("Post-Panic Test User"),
			Email: "postpanic@example.com",
		},
		Password: "password",
	}

	createdUser2, err := client.Create(ctx, "user", testUser2)
	suite.Require().NoError(err)
	defer client.Delete(ctx, createdUser2.ID.String())

	// Verify the second subscription still receives notifications
	select {
	case <-notificationChan:
		slog.Info("Second subscription working correctly after panic")
	case <-time.After(5 * time.Second):
		suite.Fail("Second subscription should still work after first handler panicked")
	}
}

// stringPtr is a helper to create string pointers for test data
func stringPtr(s string) *string {
	return &s
}
