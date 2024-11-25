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
	// Extract user ID and credits to add from the request
	userIDStr := ctx.Request.Param("id")
	creditsToAddStr := ctx.Request.Param("credits")

	// Convert userID from string to int
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err) // Return an error if the conversion fails
	}

	// Convert creditsToAdd from string to int
	creditsToAdd, err := strconv.Atoi(creditsToAddStr)
	if err != nil {
		return nil, fmt.Errorf("invalid credits to add: %w", err) // Return an error if the conversion fails
	}
	// Call the addCredits function
	message, err := AddCredits(ctx, userID, creditsToAdd)
	if err != nil {
		return nil, err
	}

	return message, nil
}
