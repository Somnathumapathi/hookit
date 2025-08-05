package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gofr.dev/pkg/gofr"
)

type CronService struct {
	app *gofr.App
}

type ScheduledWorkflow struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Schedule string `json:"schedule"` // cron expression
	Active   bool   `json:"active"`
}

func NewCronService(app *gofr.App) *CronService {
	return &CronService{app: app}
}

// StartScheduledWorkflows initializes all active scheduled workflows
func (cs *CronService) StartScheduledWorkflows(ctx *gofr.Context) error {
	// Get all workflows with schedule triggers
	workflows, err := cs.getScheduledWorkflows(ctx)
	if err != nil {
		return fmt.Errorf("failed to get scheduled workflows: %w", err)
	}

	// Register each scheduled workflow as a cron job
	for _, workflow := range workflows {
		if workflow.Active {
			cronExpr := cs.convertToCronExpression(workflow.Schedule)
			jobName := fmt.Sprintf("workflow_%d", workflow.ID)

			log.Printf("Registering cron job: %s with expression: %s", jobName, cronExpr)

			// Create a closure to capture the workflow ID
			workflowID := workflow.ID
			cs.app.AddCronJob(cronExpr, jobName, func(c *gofr.Context) {
				cs.executeScheduledWorkflow(c, workflowID)
			})
		}
	}

	return nil
}

// getScheduledWorkflows retrieves all workflows that have schedule triggers
func (cs *CronService) getScheduledWorkflows(ctx *gofr.Context) ([]ScheduledWorkflow, error) {
	query := `
		SELECT DISTINCT w.id, w.name, s.payload->>'frequency' as schedule
		FROM workflows w
		JOIN steps s ON w.id = s.workflow_id
		WHERE s.step_type = 'trigger' 
		AND s.payload->>'triggerType' = 'schedule'
		AND w.active = true
	`

	rows, err := ctx.SQL.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []ScheduledWorkflow
	for rows.Next() {
		var workflow ScheduledWorkflow
		var schedule string

		err := rows.Scan(&workflow.ID, &workflow.Name, &schedule)
		if err != nil {
			log.Printf("Error scanning workflow row: %v", err)
			continue
		}

		workflow.Schedule = schedule
		workflow.Active = true
		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// convertToCronExpression converts frequency strings to cron expressions
func (cs *CronService) convertToCronExpression(frequency string) string {
	switch frequency {
	case "hourly":
		return "0 0 * * * *" // Every hour at minute 0
	case "daily":
		return "0 0 9 * * *" // Every day at 9 AM
	case "weekly":
		return "0 0 9 * * 1" // Every Monday at 9 AM
	case "monthly":
		return "0 0 9 1 * *" // First day of every month at 9 AM
	default:
		// Try to parse as custom cron expression
		return frequency
	}
}

// executeScheduledWorkflow executes a workflow triggered by cron
func (cs *CronService) executeScheduledWorkflow(c *gofr.Context, workflowID int) {
	log.Printf("Executing scheduled workflow ID: %d", workflowID)

	// Get workflow details
	workflow, err := cs.getWorkflowByID(c, workflowID)
	if err != nil {
		c.Logger.Errorf("Failed to get workflow %d: %v", workflowID, err)
		return
	}

	// Get workflow steps
	steps, err := cs.getWorkflowSteps(c, workflowID)
	if err != nil {
		c.Logger.Errorf("Failed to get steps for workflow %d: %v", workflowID, err)
		return
	}

	// Create execution context
	executionData := map[string]interface{}{
		"trigger_type": "schedule",
		"timestamp":    time.Now(),
		"workflow_id":  workflowID,
	}

	// Execute workflow steps
	err = cs.executeWorkflowSteps(c, steps, executionData)
	if err != nil {
		c.Logger.Errorf("Failed to execute workflow %d: %v", workflowID, err)
		return
	}

	// Log successful execution
	cs.logWorkflowExecution(c, workflowID, "success", "Scheduled execution completed")
	c.Logger.Infof("Successfully executed scheduled workflow: %s (ID: %d)", workflow.Name, workflowID)
}

// getWorkflowByID retrieves a workflow by its ID
func (cs *CronService) getWorkflowByID(ctx *gofr.Context, workflowID int) (*Workflow, error) {
	query := "SELECT id, name, webhook_url FROM workflows WHERE id = $1"

	var workflow Workflow
	err := ctx.SQL.QueryRowContext(ctx, query, workflowID).Scan(
		&workflow.ID,
		&workflow.Name,
		&workflow.WebhookID,
	)

	if err != nil {
		return nil, err
	}

	return &workflow, nil
}

// getWorkflowSteps retrieves all steps for a workflow
func (cs *CronService) getWorkflowSteps(ctx *gofr.Context, workflowID int) ([]Step, error) {
	query := `
		SELECT id, workflow_id, name, step_type, payload, step_order 
		FROM steps 
		WHERE workflow_id = $1 
		ORDER BY step_order
	`

	rows, err := ctx.SQL.QueryContext(ctx, query, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []Step
	for rows.Next() {
		var step Step
		var payloadJSON []byte

		err := rows.Scan(&step.ID, &step.WorkflowID, &step.Name, &step.Type, &payloadJSON, &step.StepOrder)
		if err != nil {
			return nil, err
		}

		// Parse JSON payload
		if len(payloadJSON) > 0 {
			err = json.Unmarshal(payloadJSON, &step.Payload)
			if err != nil {
				log.Printf("Error unmarshaling payload for step %d: %v", step.ID, err)
				step.Payload = make(map[string]interface{})
			}
		}

		steps = append(steps, step)
	}

	return steps, nil
}

// executeWorkflowSteps executes all steps in a workflow
func (cs *CronService) executeWorkflowSteps(c *gofr.Context, steps []Step, data map[string]interface{}) error {
	currentData := data

	for _, step := range steps {
		log.Printf("Executing step: %s (Type: %s)", step.Name, step.Type)

		// Skip trigger steps in scheduled execution (already triggered by cron)
		if step.Type == "trigger" {
			continue
		}

		// Execute step based on type
		result, err := cs.executeStep(c, step, currentData)
		if err != nil {
			return fmt.Errorf("failed to execute step %s: %w", step.Name, err)
		}

		// Update current data with step result
		if result != nil {
			currentData = result
		}
	}

	return nil
}

// executeStep executes a single workflow step
func (cs *CronService) executeStep(c *gofr.Context, step Step, data map[string]interface{}) (map[string]interface{}, error) {
	switch step.Type {
	case "parse":
		return cs.executeParseStep(step, data)
	case "filter":
		return cs.executeFilterStep(step, data)
	case "action":
		return cs.executeActionStep(c, step, data)
	default:
		return data, nil // Pass through unknown step types
	}
}

// executeParseStep executes a data parsing step
func (cs *CronService) executeParseStep(step Step, data map[string]interface{}) (map[string]interface{}, error) {
	// Basic parsing logic - can be expanded based on parse type
	result := make(map[string]interface{})

	// Copy input data
	for k, v := range data {
		result[k] = v
	}

	// Add parsing metadata
	result["parsed_at"] = time.Now()
	result["parse_type"] = step.Payload["parseType"]

	log.Printf("Executed parse step: %s", step.Name)
	return result, nil
}

// executeFilterStep executes a data filtering step
func (cs *CronService) executeFilterStep(step Step, data map[string]interface{}) (map[string]interface{}, error) {
	// Basic filtering logic - can be expanded based on filter type
	filterType, _ := step.Payload["filterType"].(string)

	switch filterType {
	case "condition":
		// Implement condition filtering
		field, _ := step.Payload["field"].(string)
		operator, _ := step.Payload["operator"].(string)
		value := step.Payload["value"]

		log.Printf("Applying filter: %s %s %v", field, operator, value)
		// Add filter logic here

	case "validation":
		// Implement validation filtering
		log.Printf("Applying validation filter")

	default:
		log.Printf("Unknown filter type: %s", filterType)
	}

	log.Printf("Executed filter step: %s", step.Name)
	return data, nil
}

// executeActionStep executes an action step
func (cs *CronService) executeActionStep(c *gofr.Context, step Step, data map[string]interface{}) (map[string]interface{}, error) {
	actionType, _ := step.Payload["actionType"].(string)

	switch actionType {
	case "database":
		return cs.executeDatabaseAction(c, step, data)
	case "api_call":
		return cs.executeAPIAction(step, data)
	case "email":
		return cs.executeEmailAction(step, data)
	default:
		log.Printf("Unknown action type: %s", actionType)
		return data, nil
	}
}

// executeDatabaseAction executes a database action
func (cs *CronService) executeDatabaseAction(c *gofr.Context, step Step, data map[string]interface{}) (map[string]interface{}, error) {
	table, _ := step.Payload["table"].(string)
	operation, _ := step.Payload["operation"].(string)

	log.Printf("Executing database action: %s on table %s", operation, table)

	// Basic database operation - expand as needed
	switch operation {
	case "insert":
		// Insert data into table
		log.Printf("Inserting data into table: %s", table)
	case "update":
		// Update data in table
		log.Printf("Updating data in table: %s", table)
	case "upsert":
		// Upsert data into table
		log.Printf("Upserting data into table: %s", table)
	}

	return data, nil
}

// executeAPIAction executes an API call action
func (cs *CronService) executeAPIAction(step Step, data map[string]interface{}) (map[string]interface{}, error) {
	url, _ := step.Payload["url"].(string)
	method, _ := step.Payload["method"].(string)

	log.Printf("Executing API call: %s %s", method, url)

	// Implement HTTP client call here
	// For now, just log the action

	return data, nil
}

// executeEmailAction executes an email action
func (cs *CronService) executeEmailAction(step Step, data map[string]interface{}) (map[string]interface{}, error) {
	to, _ := step.Payload["to"].(string)
	subject, _ := step.Payload["subject"].(string)

	log.Printf("Sending email to: %s with subject: %s", to, subject)

	// Implement email sending here
	// For now, just log the action

	return data, nil
}

// logWorkflowExecution logs the execution result
func (cs *CronService) logWorkflowExecution(ctx *gofr.Context, workflowID int, status, message string) {
	query := `
		INSERT INTO workflow_executions (workflow_id, status, message, executed_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := ctx.SQL.ExecContext(ctx, query, workflowID, status, message, time.Now())
	if err != nil {
		log.Printf("Failed to log workflow execution: %v", err)
	}
}

// Workflow and Step structs (should be shared)
type Workflow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	WebhookID string `json:"webhook_id"`
}

type Step struct {
	ID         int                    `json:"id"`
	WorkflowID int                    `json:"workflow_id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Payload    map[string]interface{} `json:"payload"`
	StepOrder  int                    `json:"step_order"`
}
