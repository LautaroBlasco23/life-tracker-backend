package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// User & Auth metrics
	UserRegistrations = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_user_registrations_total",
		Help: "Total number of user registrations",
	})

	ActiveUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lifetracker_active_users",
		Help: "Current number of active users",
	})

	AuthAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_auth_attempts_total",
		Help: "Total authentication attempts by result",
	}, []string{"result"})

	PasswordChanges = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_password_changes_total",
		Help: "Total number of password changes",
	})

	EmailChanges = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_email_changes_total",
		Help: "Total number of email changes",
	})

	TokenRefreshes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_token_refreshes_total",
		Help: "Total number of token refreshes",
	})

	ProfileImageUploads = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_profile_image_uploads_total",
		Help: "Total number of profile image uploads",
	})

	ProfileImageDeletes = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_profile_image_deletes_total",
		Help: "Total number of profile image deletions",
	})

	// Activity metrics
	ActivitiesCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_activities_created_total",
		Help: "Total activities created by frequency",
	}, []string{"frequency"})

	ActivitiesDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_activities_deleted_total",
		Help: "Total activities deleted",
	})

	ActivitiesUpdated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_activities_updated_total",
		Help: "Total activities updated",
	})

	ActivityCompletions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_activity_completions_total",
		Help: "Total activity completions by frequency",
	}, []string{"frequency"})

	ActivityCompletionReverts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_activity_completion_reverts_total",
		Help: "Total activity completion reverts",
	})

	ActivityStreakDays = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "lifetracker_activity_streak_days",
		Help:    "Distribution of activity streak lengths in days",
		Buckets: []float64{1, 3, 7, 14, 30, 60, 90, 180, 365},
	})

	ActiveActivities = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lifetracker_active_activities",
		Help: "Current number of active activities",
	})

	ActivitiesByFrequency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lifetracker_activities_by_frequency",
		Help: "Current number of activities by frequency type",
	}, []string{"frequency"})

	ActivitiesByDayTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lifetracker_activities_by_daytime",
		Help: "Current number of activities by day time",
	}, []string{"daytime"})

	// Finance metrics
	TransactionsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_transactions_created_total",
		Help: "Total transactions created by type and category",
	}, []string{"type", "category"})

	TransactionAmount = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lifetracker_transaction_amount",
		Help:    "Transaction amounts by type and category",
		Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000, 50000},
	}, []string{"type", "category"})

	TransactionsDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_transactions_deleted_total",
		Help: "Total transactions deleted",
	})

	TransactionsUpdated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_transactions_updated_total",
		Help: "Total transactions updated",
	})

	PaymentsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_payments_created_total",
		Help: "Total payments created by transaction type",
	}, []string{"type"})

	PaymentsDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_payments_deleted_total",
		Help: "Total payments deleted",
	})

	PaymentAmount = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lifetracker_payment_amount",
		Help:    "Payment amounts by transaction type",
		Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
	}, []string{"type"})

	FixedTransactionsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lifetracker_fixed_transactions_total",
		Help: "Current number of fixed transactions by type",
	}, []string{"type"})

	TotalIncomeAmount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lifetracker_total_income_amount",
		Help: "Total income amount in the system",
	})

	TotalOutcomeAmount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lifetracker_total_outcome_amount",
		Help: "Total outcome amount in the system",
	})

	// Note metrics
	NotesCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_notes_created_total",
		Help: "Total notes created",
	})

	NotesDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_notes_deleted_total",
		Help: "Total notes deleted",
	})

	NotesUpdated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_notes_updated_total",
		Help: "Total notes updated",
	})

	ActiveNotes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lifetracker_active_notes",
		Help: "Current number of active notes",
	})

	NoteContentLength = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "lifetracker_note_content_length",
		Help:    "Distribution of note content lengths in characters",
		Buckets: []float64{100, 500, 1000, 2500, 5000, 10000, 25000},
	})

	// Time Record metrics
	TimeRecordsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_time_records_created_total",
		Help: "Total time records created by category",
	}, []string{"category"})

	TimeRecordsDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_time_records_deleted_total",
		Help: "Total time records deleted",
	})

	TimeRecordsUpdated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_time_records_updated_total",
		Help: "Total time records updated",
	})

	TimeRecordDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lifetracker_time_record_duration_minutes",
		Help:    "Distribution of time record durations in minutes by category",
		Buckets: []float64{5, 15, 30, 60, 120, 240, 480},
	}, []string{"category"})

	TotalTimeTracked = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lifetracker_total_time_tracked_minutes",
		Help: "Total minutes tracked across all users",
	})

	TimeTrackedByCategory = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lifetracker_time_tracked_by_category_minutes",
		Help: "Total minutes tracked by category",
	}, []string{"category"})

	// HTTP metrics
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lifetracker_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "endpoint", "status"})

	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_http_requests_total",
		Help: "Total HTTP requests by method, endpoint and status",
	}, []string{"method", "endpoint", "status"})

	HTTPErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_http_errors_total",
		Help: "Total HTTP errors by method and endpoint",
	}, []string{"method", "endpoint"})

	// Database metrics
	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lifetracker_db_query_duration_seconds",
		Help:    "Database query duration in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"operation", "entity"})

	DBConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lifetracker_db_connections_active",
		Help: "Number of active database connections",
	})
)
