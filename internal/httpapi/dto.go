package httpapi

import (
	"fmt"
	"strings"
	"time"

	"task-ef-mobile/internal/subscriptions"

	"github.com/google/uuid"
)

const monthLayout = "01-2006"

type subscriptionRequest struct {
	ServiceName string `json:"service_name"`
	Price       int    `json:"price"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date,omitempty"`
}

type subscriptionResponse struct {
	ID          string `json:"id"`
	ServiceName string `json:"service_name"`
	Price       int    `json:"price"`
	UserID      string `json:"user_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type totalResponse struct {
	Total int64 `json:"total"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (r subscriptionRequest) toInput() (subscriptions.CreateInput, error) {
	serviceName := strings.TrimSpace(r.ServiceName)
	if serviceName == "" {
		return subscriptions.CreateInput{}, fmt.Errorf("service_name is required")
	}
	if r.Price < 0 {
		return subscriptions.CreateInput{}, fmt.Errorf("price must be greater than or equal to 0")
	}

	userID, err := uuid.Parse(r.UserID)
	if err != nil {
		return subscriptions.CreateInput{}, fmt.Errorf("user_id must be a valid UUID")
	}

	startMonth, err := parseMonth(r.StartDate)
	if err != nil {
		return subscriptions.CreateInput{}, fmt.Errorf("start_date must use MM-YYYY format")
	}

	var endMonth *time.Time
	if strings.TrimSpace(r.EndDate) != "" {
		parsedEnd, err := parseMonth(r.EndDate)
		if err != nil {
			return subscriptions.CreateInput{}, fmt.Errorf("end_date must use MM-YYYY format")
		}
		if parsedEnd.Before(startMonth) {
			return subscriptions.CreateInput{}, fmt.Errorf("end_date must be greater than or equal to start_date")
		}
		endMonth = &parsedEnd
	}

	return subscriptions.CreateInput{
		ServiceName: serviceName,
		Price:       r.Price,
		UserID:      userID,
		StartMonth:  startMonth,
		EndMonth:    endMonth,
	}, nil
}

func toResponse(sub subscriptions.Subscription) subscriptionResponse {
	response := subscriptionResponse{
		ID:          sub.ID.String(),
		ServiceName: sub.ServiceName,
		Price:       sub.Price,
		UserID:      sub.UserID.String(),
		StartDate:   formatMonth(sub.StartMonth),
		CreatedAt:   sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   sub.UpdatedAt.Format(time.RFC3339),
	}
	if sub.EndMonth != nil {
		response.EndDate = formatMonth(*sub.EndMonth)
	}
	return response
}

func parseMonth(value string) (time.Time, error) {
	parsed, err := time.Parse(monthLayout, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func formatMonth(value time.Time) string {
	return value.UTC().Format(monthLayout)
}
