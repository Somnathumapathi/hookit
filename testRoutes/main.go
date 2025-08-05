package testRoutes

import (
	"fmt"
	"strconv"

	"gofr.dev/pkg/gofr"
)

// CreateTestScheduledWorkflow creates a test workflow with schedule trigger for demonstration
func CreateTestScheduledWorkflow(ctx *gofr.Context) (interface{}, error) {
	// First create a test workflow (using the existing schema)
	workflowQuery := `
		INSERT INTO workflows (name, webhook_url, user_id) 
		VALUES ($1, $2, $3) 
		RETURNING id
	`

	var workflowID int
	err := ctx.SQL.QueryRowContext(ctx, workflowQuery,
		"Test Scheduled Workflow",
		"test-scheduled-workflow",
		2, // using the registered test user ID
	).Scan(&workflowID)

	if err != nil {
		return nil, fmt.Errorf("failed to create test workflow: %w", err)
	}

	// Create schedule trigger step
	triggerPayload := `{
		"triggerType": "schedule",
		"frequency": "hourly",
		"time": "00:00",
		"timezone": "UTC"
	}`

	stepQuery := `
		INSERT INTO steps (workflow_id, name, type, payload, step_order)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = ctx.SQL.ExecContext(ctx, stepQuery, workflowID, "Schedule Trigger", "trigger", triggerPayload, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to create trigger step: %w", err)
	}

	// Create a simple action step
	actionPayload := `{
		"actionType": "api_call",
		"url": "https://httpbin.org/post",
		"method": "POST"
	}`

	_, err = ctx.SQL.ExecContext(ctx, stepQuery, workflowID, "Test API Call", "action", actionPayload, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to create action step: %w", err)
	}

	return map[string]interface{}{
		"message":     "Test scheduled workflow created successfully",
		"workflow_id": workflowID,
		"webhook_url": "test-scheduled-workflow",
		"schedule":    "Every hour at 00:00 UTC",
	}, nil
}

// TestCronExecution manually tests the cron execution logic
func TestCronExecution(ctx *gofr.Context) (interface{}, error) {
	workflowIDStr := ctx.PathParam("workflowId")
	if workflowIDStr == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	workflowID, err := strconv.Atoi(workflowIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}

	// Simulate a scheduled execution log entry
	logQuery := `
		INSERT INTO workflow_executions (workflow_id, status, message, executed_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, executed_at
	`

	var executionID int
	var executedAt string
	err = ctx.SQL.QueryRowContext(ctx, logQuery, workflowID, "success", "Manual test execution").Scan(&executionID, &executedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to log test execution: %w", err)
	}

	return map[string]interface{}{
		"message":      "Test execution logged successfully",
		"execution_id": executionID,
		"workflow_id":  workflowID,
		"executed_at":  executedAt,
		"status":       "success",
	}, nil
}

// GetCronStatus checks the current status of cron jobs and scheduled workflows
func GetCronStatus(ctx *gofr.Context) (interface{}, error) {
	// Check for scheduled workflows
	scheduledQuery := `
		SELECT COUNT(*) as scheduled_count
		FROM workflows w
		JOIN steps s ON w.id = s.workflow_id
		WHERE s.step_type = 'trigger' 
		AND s.payload->>'triggerType' = 'schedule'
		AND w.active = true
	`

	var scheduledCount int
	err := ctx.SQL.QueryRowContext(ctx, scheduledQuery).Scan(&scheduledCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count scheduled workflows: %w", err)
	}

	// Check recent executions
	executionsQuery := `
		SELECT COUNT(*) as execution_count
		FROM workflow_executions
		WHERE executed_at > NOW() - INTERVAL '24 hours'
	`

	var executionCount int
	err = ctx.SQL.QueryRowContext(ctx, executionsQuery).Scan(&executionCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count recent executions: %w", err)
	}

	// Check if workflow_executions table exists
	tableExistsQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'workflow_executions'
		)
	`

	var tableExists bool
	err = ctx.SQL.QueryRowContext(ctx, tableExistsQuery).Scan(&tableExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check table existence: %w", err)
	}

	return map[string]interface{}{
		"scheduled_workflows": scheduledCount,
		"recent_executions":   executionCount,
		"table_exists":        tableExists,
		"message":             "Cron system status retrieved successfully",
	}, nil
}
