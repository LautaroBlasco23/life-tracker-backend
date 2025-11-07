package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
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

	ActivitiesCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_activities_created_total",
		Help: "Total activities created by frequency",
	}, []string{"frequency"})

	ActivitiesDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lifetracker_activities_deleted_total",
		Help: "Total activities deleted",
	})

	ActivityCompletions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_activity_completions_total",
		Help: "Total activity completions by frequency",
	}, []string{"frequency"})

	ActivityStreakDays = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "lifetracker_activity_streak_days",
		Help:    "Distribution of activity streak lengths in days",
		Buckets: []float64{1, 3, 7, 14, 30, 60, 90, 180, 365},
	})

	TransactionsCreated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lifetracker_transactions_created_total",
		Help: "Total transactions created by type",
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
)
