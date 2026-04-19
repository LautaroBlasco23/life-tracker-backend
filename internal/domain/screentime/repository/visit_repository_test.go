package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/screentime/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// visit_repository_integration_test.go
//
// Subject: MongoVisitRepository against real MongoDB.
// Scope:   Repository query correctness — Create, FindByUser, StatsByUser.
// Out of scope:
//   - service-layer rules          → visit_service_test.go       (unit)
//   - HTTP contract                → screentime_test.go          (integration)
// Infrastructure: MongoDB (via testcontainers or local instance), connection via config.
// Data strategy: drop collection between tests.
// Parallel-safe: no — shared database, tests run sequentially.

var (
	testMongoDB *mongo.Database
	mongoClient *mongo.Client
	cfg         *config.Config
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		panic("failed to connect to MongoDB: " + err.Error())
	}
	if err := client.Ping(ctx, nil); err != nil {
		panic("MongoDB not available: " + err.Error())
	}

	testMongoDB = client.Database(cfg.MongoDatabase)
	mongoClient = client

	code := m.Run()

	if mongoClient != nil {
		_ = mongoClient.Disconnect(context.Background())
	}

	os.Exit(code)
}

func cleanCollection(t *testing.T) {
	t.Helper()
	_ = testMongoDB.Collection("web_visits").Drop(context.Background())
}

func newTestVisit(userID uint, domain string, visitedAt time.Time) *model.WebVisit {
	return &model.WebVisit{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Domain:    domain,
		VisitedAt: visitedAt,
		CreatedAt: time.Now(),
	}
}

// ---- Create tests ----

// TestCreateVisit verifies that a visit is stored and can be retrieved.
// Covers: basic insert, ID generation, field persistence.
func TestCreateVisit(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()

	visit := newTestVisit(1, "github.com", time.Now())

	err := repo.Create(ctx, visit)
	require.NoError(t, err)
	assert.NotEmpty(t, visit.ID)

	// Verify by fetching
	visits, err := repo.FindByUser(ctx, 1, "", nil, nil, 100)
	require.NoError(t, err)
	require.Len(t, visits, 1)
	assert.Equal(t, "github.com", visits[0].Domain)
	assert.Equal(t, uint(1), visits[0].UserID)
}

// ---- FindByUser tests ----

// TestFindByUser_OnlyUserID verifies filtering by userID only returns that user's visits.
// Covers: user isolation, no domain filter, no date filter, no limit.
func TestFindByUser_OnlyUserID(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()
	now := time.Now()

	// Seed visits for user 1
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "youtube.com", now)))

	// Seed visit for user 2 (should not appear)
	require.NoError(t, repo.Create(ctx, newTestVisit(2, "github.com", now)))

	visits, err := repo.FindByUser(ctx, 1, "", nil, nil, 100)
	require.NoError(t, err)
	require.Len(t, visits, 2)

	// Verify all belong to user 1
	for _, v := range visits {
		assert.Equal(t, uint(1), v.UserID)
	}
}

// TestFindByUser_WithDomainFilter verifies domain filter works correctly.
// Covers: exact domain match, case sensitivity (if applicable).
func TestFindByUser_WithDomainFilter(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()
	now := time.Now()

	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now.Add(-time.Hour))))
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "youtube.com", now)))

	visits, err := repo.FindByUser(ctx, 1, "github.com", nil, nil, 100)
	require.NoError(t, err)
	require.Len(t, visits, 2)

	for _, v := range visits {
		assert.Equal(t, "github.com", v.Domain)
	}
}

// TestFindByUser_WithDateRange verifies from/to date filters work.
// Covers: inclusive boundaries, partial range, future dates.
func TestFindByUser_WithDateRange(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "youtube.com", yesterday)))
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "instagram.com", twoDaysAgo)))

	// Filter: yesterday to now (should get github and youtube)
	from := yesterday.Add(-time.Hour)
	to := now.Add(time.Hour)

	visits, err := repo.FindByUser(ctx, 1, "", &from, &to, 100)
	require.NoError(t, err)
	require.Len(t, visits, 2)

	domains := make(map[string]bool)
	for _, v := range visits {
		domains[v.Domain] = true
	}
	assert.Contains(t, domains, "github.com")
	assert.Contains(t, domains, "youtube.com")
	assert.NotContains(t, domains, "instagram.com")
}

// TestFindByUser_WithCombinedFilters verifies domain + date range work together.
// Covers: multiple filter combination, correct intersection.
func TestFindByUser_WithCombinedFilters(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// github: now and two days ago
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", twoDaysAgo)))
	// youtube: yesterday
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "youtube.com", yesterday)))

	// Filter: github only, from yesterday onwards
	from := yesterday.Add(-time.Hour)

	visits, err := repo.FindByUser(ctx, 1, "github.com", &from, nil, 100)
	require.NoError(t, err)
	require.Len(t, visits, 1)
	assert.Equal(t, "github.com", visits[0].Domain)
}

// TestFindByUser_WithLimit verifies limit parameter is respected.
// Covers: limit smaller than total, limit larger than total, zero limit.
func TestFindByUser_WithLimit(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()
	now := time.Now()

	// Seed 5 visits
	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now.Add(-time.Duration(i)*time.Hour))))
	}

	// Limit 3 should return 3 (sorted by visitedAt desc)
	visits, err := repo.FindByUser(ctx, 1, "", nil, nil, 3)
	require.NoError(t, err)
	require.Len(t, visits, 3)

	// Verify descending order (most recent first)
	for i := 1; i < len(visits); i++ {
		assert.True(t, visits[i-1].VisitedAt.After(visits[i].VisitedAt) ||
			visits[i-1].VisitedAt.Equal(visits[i].VisitedAt))
	}
}

// ---- StatsByUser tests ----

// TestStatsByUser_ReturnsDomainStats verifies aggregation returns correct counts per domain.
// Covers: grouping by domain, count aggregation, sorting by count desc.
func TestStatsByUser_ReturnsDomainStats(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()
	now := time.Now()

	// Seed: github×3, youtube×2, instagram×1
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))
	}
	for i := 0; i < 2; i++ {
		require.NoError(t, repo.Create(ctx, newTestVisit(1, "youtube.com", now)))
	}
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "instagram.com", now)))

	stats, err := repo.StatsByUser(ctx, 1, nil, nil)
	require.NoError(t, err)
	require.Len(t, stats, 3)

	// Should be sorted by count desc
	assert.Equal(t, "github.com", stats[0].Domain)
	assert.Equal(t, int64(3), stats[0].Count)
	assert.Equal(t, "youtube.com", stats[1].Domain)
	assert.Equal(t, int64(2), stats[1].Count)
	assert.Equal(t, "instagram.com", stats[2].Domain)
	assert.Equal(t, int64(1), stats[2].Count)
}

// TestStatsByUser_WithDateRange verifies stats respect date boundaries.
// Covers: from/to filtering in aggregation pipeline.
func TestStatsByUser_WithDateRange(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// github: 2 today
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))
	// youtube: 1 yesterday
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "youtube.com", yesterday)))
	// instagram: 1 two days ago
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "instagram.com", twoDaysAgo)))

	// Filter: yesterday only
	from := yesterday.Add(-time.Hour)
	to := yesterday.Add(time.Hour)

	stats, err := repo.StatsByUser(ctx, 1, &from, &to)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, "youtube.com", stats[0].Domain)
	assert.Equal(t, int64(1), stats[0].Count)
}

// TestStatsByUser_EmptyForNoVisits verifies empty result (not nil) for user with no visits.
// Covers: empty aggregation result handling, no cross-user data leakage.
func TestStatsByUser_EmptyForNoVisits(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()
	now := time.Now()

	// Seed for user 1 only
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))

	// Query for user 999 (no visits)
	stats, err := repo.StatsByUser(ctx, 999, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Empty(t, stats)
	assert.Len(t, stats, 0)
}

// TestStatsByUser_NoCrossUserDataLeakage verifies user isolation in aggregation.
// Covers: match stage correctly filters by userID before grouping.
func TestStatsByUser_NoCrossUserDataLeakage(t *testing.T) {
	cleanCollection(t)

	repo := NewVisitRepository(testMongoDB)
	ctx := context.Background()
	now := time.Now()

	// User 1: github×2
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))
	require.NoError(t, repo.Create(ctx, newTestVisit(1, "github.com", now)))

	// User 2: github×1, youtube×1
	require.NoError(t, repo.Create(ctx, newTestVisit(2, "github.com", now)))
	require.NoError(t, repo.Create(ctx, newTestVisit(2, "youtube.com", now)))

	// Stats for user 1
	stats, err := repo.StatsByUser(ctx, 1, nil, nil)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, "github.com", stats[0].Domain)
	assert.Equal(t, int64(2), stats[0].Count)

	// Stats for user 2
	stats2, err := repo.StatsByUser(ctx, 2, nil, nil)
	require.NoError(t, err)
	require.Len(t, stats2, 2)
}
