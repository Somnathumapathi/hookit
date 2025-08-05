package cronRoutes

import (
	"fmt"
	"strconv"

	"gofr.dev/pkg/gofr"
)

// GetScheduledWorkflows returns all workflows that have schedule triggers
func GetScheduledWorkflows(ctx *gofr.Context) (interface{}, error) {
	query := `
		SELECT DISTINCT w.id, w.name, s.payload->>'frequency' as frequency,
		       s.payload->>'time' as schedule_time, w.created_at
		FROM workflows w
		JOIN steps s ON w.id = s.workflow_id
		WHERE s.step_type = 'trigger' 
		AND s.payload->>'triggerType' = 'schedule'
		ORDER BY w.created_at DESC
	`

	rows, err := ctx.SQL.QueryContext(ctx, query)
	if err != nil {
		ctx.Logger.Errorf("Error querying scheduled workflows: %v", err)
		return nil, fmt.Errorf("failed to get scheduled workflows: %w", err)
	}
	defer rows.Close()

	type ScheduledWorkflowInfo struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		Frequency    string `json:"frequency"`
		ScheduleTime string `json:"schedule_time"`
		CreatedAt    string `json:"created_at"`
	}

	var workflows []ScheduledWorkflowInfo
	for rows.Next() {
		var workflow ScheduledWorkflowInfo
		var scheduleTime *string

		err := rows.Scan(
			&workflow.ID,
			&workflow.Name,
			&workflow.Frequency,
			&scheduleTime,
			&workflow.CreatedAt,
		)
		if err != nil {
			ctx.Logger.Errorf("Error scanning workflow row: %v", err)
			continue
		}

		if scheduleTime != nil {
			workflow.ScheduleTime = *scheduleTime
		}

		workflows = append(workflows, workflow)
	}

	return map[string]interface{}{
		"workflows": workflows,
		"count":     len(workflows),
	}, nil
}

// GetWorkflowExecutions returns execution history for a specific workflow
func GetWorkflowExecutions(ctx *gofr.Context) (interface{}, error) {
	workflowIDStr := ctx.PathParam("workflowId")
	workflowID, err := strconv.Atoi(workflowIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}

	query := `
		SELECT id, workflow_id, status, message, executed_at, duration_ms
		FROM workflow_executions
		WHERE workflow_id = $1
		ORDER BY executed_at DESC
		LIMIT 50
	`

	rows, err := ctx.SQL.QueryContext(ctx, query, workflowID)
	if err != nil {
		ctx.Logger.Errorf("Error querying workflow executions: %v", err)
		return nil, fmt.Errorf("failed to get workflow executions: %w", err)
	}
	defer rows.Close()

	type WorkflowExecution struct {
		ID         int     `json:"id"`
		WorkflowID int     `json:"workflow_id"`
		Status     string  `json:"status"`
		Message    *string `json:"message"`
		ExecutedAt string  `json:"executed_at"`
		DurationMs *int    `json:"duration_ms"`
	}

	var executions []WorkflowExecution
	for rows.Next() {
		var execution WorkflowExecution

		err := rows.Scan(
			&execution.ID,
			&execution.WorkflowID,
			&execution.Status,
			&execution.Message,
			&execution.ExecutedAt,
			&execution.DurationMs,
		)
		if err != nil {
			ctx.Logger.Errorf("Error scanning execution row: %v", err)
			continue
		}

		executions = append(executions, execution)
	}

	return map[string]interface{}{
		"executions":  executions,
		"count":       len(executions),
		"workflow_id": workflowID,
	}, nil
}

// ToggleWorkflowSchedule enables/disables scheduled execution for a workflow
func ToggleWorkflowSchedule(ctx *gofr.Context) (interface{}, error) {
	workflowIDStr := ctx.PathParam("workflowId")
	workflowID, err := strconv.Atoi(workflowIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}

	// Parse request body to get the active status
	var requestBody struct {
		Active bool `json:"active"`
	}

	if err := ctx.Bind(&requestBody); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	query := `UPDATE workflows SET active = $1 WHERE id = $2`

	_, err = ctx.SQL.ExecContext(ctx, query, requestBody.Active, workflowID)
	if err != nil {
		ctx.Logger.Errorf("Error updating workflow active status: %v", err)
		return nil, fmt.Errorf("failed to update workflow schedule status: %w", err)
	}

	status := "disabled"
	if requestBody.Active {
		status = "enabled"
	}

	return map[string]interface{}{
		"message":     "Workflow schedule " + status + " successfully",
		"workflow_id": workflowID,
		"active":      requestBody.Active,
	}, nil
}
