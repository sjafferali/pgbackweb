package destinations

import (
	"database/sql"
	"time"

	"github.com/eduardolat/pgbackweb/internal/config"
	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/google/uuid"
)

type handlers struct {
	servs *dbgen.Queries
	env   config.Env
}

func New(servs *dbgen.Queries, env config.Env) *handlers {
	return &handlers{servs: servs, env: env}
}

type createDestinationRequest struct {
	Name       string `json:"name" validate:"required"`
	BucketName string `json:"bucket_name" validate:"required"`
	AccessKey  string `json:"access_key" validate:"required"`
	SecretKey  string `json:"secret_key" validate:"required"`
	Region     string `json:"region" validate:"required"`
	Endpoint   string `json:"endpoint" validate:"required"`
}

type destinationResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	BucketName string    `json:"bucket_name"`
	Region     string    `json:"region"`
	Endpoint   string    `json:"endpoint"`
	TestOk     *bool     `json:"test_ok,omitempty"`
	TestError  *string   `json:"test_error,omitempty"`
	LastTestAt *string   `json:"last_test_at,omitempty"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  *string   `json:"updated_at,omitempty"`
}

func toDestinationResponse(dest interface{}) destinationResponse {
	var id uuid.UUID
	var name, bucketName, region, endpoint string
	var testOk sql.NullBool
	var testError sql.NullString
	var lastTestAt, updatedAt sql.NullTime
	var createdAt time.Time

	switch d := dest.(type) {
	case dbgen.DestinationsServiceGetDestinationRow:
		id = d.ID
		name = d.Name
		bucketName = d.BucketName
		region = d.Region
		endpoint = d.Endpoint
		testOk = d.TestOk
		testError = d.TestError
		lastTestAt = d.LastTestAt
		updatedAt = d.UpdatedAt
		createdAt = d.CreatedAt
	case dbgen.DestinationsServiceGetAllDestinationsRow:
		id = d.ID
		name = d.Name
		bucketName = d.BucketName
		region = d.Region
		endpoint = d.Endpoint
		testOk = d.TestOk
		testError = d.TestError
		lastTestAt = d.LastTestAt
		updatedAt = d.UpdatedAt
		createdAt = d.CreatedAt
	default:
		panic("unsupported destination type")
	}

	var testOkPtr *bool
	if testOk.Valid {
		testOkPtr = &testOk.Bool
	}

	var testErrorPtr *string
	if testError.Valid {
		testErrorPtr = &testError.String
	}

	var lastTestAtPtr *string
	if lastTestAt.Valid {
		lastTestAtStr := lastTestAt.Time.Format("2006-01-02T15:04:05Z07:00")
		lastTestAtPtr = &lastTestAtStr
	}

	var updatedAtPtr *string
	if updatedAt.Valid {
		updatedAtStr := updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		updatedAtPtr = &updatedAtStr
	}

	return destinationResponse{
		ID:         id,
		Name:       name,
		BucketName: bucketName,
		Region:     region,
		Endpoint:   endpoint,
		TestOk:     testOkPtr,
		TestError:  testErrorPtr,
		LastTestAt: lastTestAtPtr,
		CreatedAt:  createdAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  updatedAtPtr,
	}
}
