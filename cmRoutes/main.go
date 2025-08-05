package cmRoutes

import (
	"fmt"
	"strconv"

	"gofr.dev/pkg/gofr"
)

func AddCredits(ctx *gofr.Context, userID int, creditsToAdd int) (string, error) {
	// Validate the number of credits to add
	if creditsToAdd <= 0 {
		return "", fmt.Errorf("credits to add must be greater than zero")
	}

	// Update the user's credits in the database
	query := `UPDATE users SET credits = credits + $1 WHERE id = $2`
	result, err := ctx.SQL.ExecContext(ctx, query, creditsToAdd, userID)
	if err != nil {
		return "", fmt.Errorf("failed to add credits: %w", err)
	}

	// Ensure that a row was updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("could not verify rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return "", fmt.Errorf("user with id %d not found", userID)
	}

	// Retrieve the updated credit count for confirmation
	var updatedCredits int
	err = ctx.SQL.QueryRowContext(ctx, `SELECT credits FROM users WHERE id = $1`, userID).Scan(&updatedCredits)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve updated credits: %w", err)
	}

	return fmt.Sprintf("Credits successfully added. User now has %d credits.", updatedCredits), nil
}

func AddCreditsHandler(ctx *gofr.Context) (interface{}, error) {
	// Parse request body to get user ID and credits
	var requestBody struct {
		UserID  int `json:"userId"`
		Credits int `json:"credits"`
	}

	if err := ctx.Bind(&requestBody); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	// Validate inputs
	if requestBody.UserID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}
	if requestBody.Credits <= 0 {
		return nil, fmt.Errorf("credits must be greater than zero")
	}

	// Call the addCredits function
	message, err := AddCredits(ctx, requestBody.UserID, requestBody.Credits)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"message":      message,
		"userId":       requestBody.UserID,
		"creditsAdded": requestBody.Credits,
	}, nil
}

// GetUserCredits returns the current credit balance for a user
func GetUserCredits(ctx *gofr.Context) (interface{}, error) {
	userIDStr := ctx.Request.PathParam("userId")
	if userIDStr == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var credits int
	var name string
	var email string

	query := `SELECT name, email, credits FROM users WHERE id = $1`
	err = ctx.SQL.QueryRowContext(ctx, query, userID).Scan(&name, &email, &credits)
	if err != nil {
		return nil, fmt.Errorf("failed to get user credits: %w", err)
	}

	return map[string]interface{}{
		"userId":  userID,
		"name":    name,
		"email":   email,
		"credits": credits,
	}, nil
}
