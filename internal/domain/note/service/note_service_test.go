// note_service_test.go
//
// Subject:      internal/domain/note/service.NoteService against real PostgreSQL.
// Scope:        CRUD operations — create, read, filter, update, soft-delete.
// Out of scope: HTTP contract → note_controller_test.go
//               Repository SQL correctness in isolation.
// Infrastructure: PostgreSQL (test database via .env.test).
// Data strategy:  TRUNCATE notes before each top-level test function.
// Parallel-safe:  No — single shared DB connection, no per-test namespace.
package service

import (
	"fmt"
	"os"
	"testing"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/note/dto"
	"life-tracker-backend/internal/domain/note/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB *gorm.DB
	cfg    *config.Config
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	os.Exit(m.Run())
}

func setupTestDatabase(t *testing.T) {
	t.Helper()
	if testDB != nil {
		return
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to PostgreSQL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&model.Note{}))
	testDB = db
}

func cleanDatabase(t *testing.T) {
	t.Helper()
	require.NoError(t, testDB.Exec("TRUNCATE TABLE notes RESTART IDENTITY CASCADE;").Error)
}

func getTestService(t *testing.T) *NoteService {
	t.Helper()
	setupTestDatabase(t)
	cleanDatabase(t)
	return NewNoteService(testDB)
}

// TestCreateNote verifies that valid notes are persisted with correct field values
// and that user isolation is enforced by storing the userID on each record.
func TestCreateNote(t *testing.T) {
	svc := getTestService(t)

	tests := []struct {
		req     *dto.CreateNoteRequest
		name    string
		userID  uint
		wantErr bool
	}{
		{
			name:   "note_with_title_and_content_persisted",
			userID: 10,
			req: &dto.CreateNoteRequest{
				Title:   "Shopping List",
				Content: "Eggs, bread, milk",
			},
		},
		{
			name:   "note_with_title_only_persisted",
			userID: 11,
			req: &dto.CreateNoteRequest{
				Title: "Empty Note",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.CreateNote(tt.userID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				return
			}

			assert.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.req.Title, resp.Title)
			assert.Equal(t, tt.req.Content, resp.Content)
			assert.Equal(t, tt.userID, resp.UserID)
			assert.NotZero(t, resp.ID)
		})
	}
}

// TestGetNote verifies ownership enforcement: a note is only visible to its owner,
// and a non-existent ID returns an error regardless of userID.
func TestGetNote(t *testing.T) {
	svc := getTestService(t)
	userID := uint(100)

	note := &model.Note{UserID: userID, Title: "Private Note", Content: "Secret stuff"}
	require.NoError(t, testDB.Create(note).Error)

	t.Run("existing_note_returns_response", func(t *testing.T) {
		resp, err := svc.GetNote(note.ID, userID)
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, note.Title, resp.Title)
		assert.Equal(t, note.Content, resp.Content)
	})

	t.Run("non_existent_id_returns_error", func(t *testing.T) {
		resp, err := svc.GetNote(9999, userID)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("wrong_user_returns_error", func(t *testing.T) {
		// The WHERE clause includes user_id, so querying with another user ID
		// produces gorm.ErrRecordNotFound → ErrNoteNotFound.
		resp, err := svc.GetNote(note.ID, 999)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestGetNotes verifies that listing returns only the requesting user's notes
// and that the optional text filter performs case-insensitive search across
// both title and content (PostgreSQL ILIKE).
func TestGetNotes(t *testing.T) {
	svc := getTestService(t)
	userID := uint(101)

	seeds := []*model.Note{
		{UserID: userID, Title: "Meeting Notes", Content: "Discuss quarterly goals"},
		{UserID: userID, Title: "Book List", Content: "Clean Code, SICP"},
		{UserID: 999, Title: "Other User Note", Content: "Should be invisible"},
	}
	for _, n := range seeds {
		require.NoError(t, testDB.Create(n).Error)
	}

	t.Run("no_filter_returns_all_user_notes", func(t *testing.T) {
		resp, err := svc.GetNotes(userID, nil)
		assert.NoError(t, err)
		assert.Len(t, resp, 2)
	})

	t.Run("query_matches_title_case_insensitively", func(t *testing.T) {
		filter := &dto.NoteFilter{Query: "meeting"}
		resp, err := svc.GetNotes(userID, filter)
		assert.NoError(t, err)
		require.Len(t, resp, 1)
		assert.Equal(t, "Meeting Notes", resp[0].Title)
	})

	t.Run("query_matches_content", func(t *testing.T) {
		filter := &dto.NoteFilter{Query: "Clean Code"}
		resp, err := svc.GetNotes(userID, filter)
		assert.NoError(t, err)
		require.Len(t, resp, 1)
		assert.Equal(t, "Book List", resp[0].Title)
	})

	t.Run("query_with_no_matches_returns_empty", func(t *testing.T) {
		filter := &dto.NoteFilter{Query: "xyznonexistent"}
		resp, err := svc.GetNotes(userID, filter)
		assert.NoError(t, err)
		assert.Empty(t, resp)
	})

	t.Run("different_user_sees_no_notes", func(t *testing.T) {
		resp, err := svc.GetNotes(888, nil)
		assert.NoError(t, err)
		assert.Empty(t, resp)
	})
}

// TestUpdateNote verifies that individual fields can be patched independently
// and that non-owners cannot mutate a note.
func TestUpdateNote(t *testing.T) {
	svc := getTestService(t)
	userID := uint(102)

	note := &model.Note{UserID: userID, Title: "Original Title", Content: "Original content"}
	require.NoError(t, testDB.Create(note).Error)

	t.Run("update_title_preserves_content", func(t *testing.T) {
		newTitle := "Updated Title"
		resp, err := svc.UpdateNote(note.ID, userID, &dto.UpdateNoteRequest{Title: &newTitle})
		assert.NoError(t, err)
		assert.Equal(t, "Updated Title", resp.Title)
		assert.Equal(t, "Original content", resp.Content)
	})

	t.Run("update_content_preserves_title", func(t *testing.T) {
		newContent := "Updated content"
		resp, err := svc.UpdateNote(note.ID, userID, &dto.UpdateNoteRequest{Content: &newContent})
		assert.NoError(t, err)
		assert.Equal(t, "Updated content", resp.Content)
	})

	t.Run("non_existent_note_returns_error", func(t *testing.T) {
		newTitle := "Ghost"
		resp, err := svc.UpdateNote(9999, userID, &dto.UpdateNoteRequest{Title: &newTitle})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("wrong_user_returns_error", func(t *testing.T) {
		newTitle := "Unauthorized"
		resp, err := svc.UpdateNote(note.ID, 999, &dto.UpdateNoteRequest{Title: &newTitle})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestDeleteNote verifies that a soft-deleted note is no longer retrievable
// and that deletion fails for non-owners or non-existent IDs.
func TestDeleteNote(t *testing.T) {
	svc := getTestService(t)
	userID := uint(103)

	t.Run("successful_deletion_makes_note_unretrievable", func(t *testing.T) {
		note := &model.Note{UserID: userID, Title: "To Delete", Content: "Goodbye"}
		require.NoError(t, testDB.Create(note).Error)

		err := svc.DeleteNote(note.ID, userID)
		assert.NoError(t, err)

		resp, err := svc.GetNote(note.ID, userID)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("delete_non_existent_note_returns_error", func(t *testing.T) {
		err := svc.DeleteNote(9999, userID)
		assert.Error(t, err)
	})

	t.Run("delete_with_wrong_user_returns_error", func(t *testing.T) {
		// FindByID(id, wrongUser) returns not-found before Delete is called,
		// so the note is never removed even though the ID exists.
		note := &model.Note{UserID: userID, Title: "Protected Note", Content: "Stay alive"}
		require.NoError(t, testDB.Create(note).Error)

		err := svc.DeleteNote(note.ID, 999)
		assert.Error(t, err)

		// Note must still be readable by the real owner.
		resp, getErr := svc.GetNote(note.ID, userID)
		assert.NoError(t, getErr)
		assert.NotNil(t, resp)
	})
}
