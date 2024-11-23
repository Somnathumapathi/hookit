package workflowRoutes

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github/Somnathumapathi/gofrhack/models"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"gofr.dev/pkg/gofr"
)

type Workflow struct {
	WebookUrl string      `json:"webhookUrl"`
	Id        int         `json:"id"`
	Steps     []Step      `json:"steps"`
	Name      string      `json:"name"`
	User      models.User `json:"users"`
}

type Step struct {
	ID        int               `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Payload   map[string]string `json:"payload"`
	StepOrder int               `json:"stepOrder"`
}

func GenerateWebhookUrl() (string, error) {
	// Generate a random 32-byte slice
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// Encode the random bytes to a base64 string
	return base64.StdEncoding.EncodeToString(b), nil
}

func CreateWorkflow(ctx *gofr.Context) (interface{}, error) {
	var workflow Workflow
	uid := ctx.Request.Param("uid")
	err := ctx.Bind(&workflow)
	if err != nil {
		return nil, err
	}
	webhookUrl, webhookUrlErr := GenerateWebhookUrl()
	if webhookUrlErr != nil {
		return nil, webhookUrlErr
	}

	query := `INSERT INTO workflows (name, webhook_url, user_id) VALUES ($1, $2, $3) RETURNING id`
	err = ctx.SQL.QueryRowContext(ctx, query, workflow.Name, webhookUrl, uid).Scan(&workflow.Id, &workflow.User)
	if err != nil {
		return nil, err
	}
	for _, step := range workflow.Steps {
		// Convert the map[string]string payload to JSON
		payloadJSON, jsonErr := json.Marshal(step.Payload)
		if jsonErr != nil {
			return nil, jsonErr
		}

		stepQuery := `INSERT INTO steps (workflow_id, name, step_type, payload, step_order)
                      VALUES ($1, $2, $3, $4, $5)`
		_, err := ctx.SQL.ExecContext(ctx, stepQuery, workflow.Id, step.Name, step.Type, payloadJSON, step.StepOrder)
		if err != nil {
			return nil, err
		}
	}
	workflow.WebookUrl = webhookUrl
	return workflow, nil
}

// func GetWorkflow(ctx *gofr.Context) (interface{}, error) {
// 	// userId := ctx.Value("uid")
// 	workflowId := ctx.Request.PathParam("id")
// 	workflowQuery := `SELECT * FROM workflows WHERE id = $1`
// 	// var workflow Workflow
// 	rows, qerr := ctx.SQL.QueryContext(ctx, workflowQuery, workflowId)
// 	if qerr != nil {
// 		return nil, qerr
// 	}
// 	var workflows []Workflow
// 	for rows.Next() {
// 		var workflow Workflow
// 		if err := rows.Scan(&workflow.Id, &workflow.Name, &workflow.WebookUrl, &workflow.User); err != nil {
// 			return nil, err
// 		}
// 		workflows = append(workflows, workflow)
// 	}
// 	response := map[string]interface{}{
// 		"me": workflows[0],
// 	}
// 	return response, nil
// }

func UpdateWorkflow(ctx *gofr.Context) (interface{}, error) {
	var workflow Workflow

	// Parse the request body to extract workflow data
	err := ctx.Bind(&workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to bind workflow: %w", err)
	}

	// Update the workflow's name and webhook URL
	updateWorkflowQuery := `UPDATE workflows SET name = $1, webhook_url = $2 WHERE id = $3`
	_, err = ctx.SQL.ExecContext(ctx, updateWorkflowQuery, workflow.Name, workflow.WebookUrl, workflow.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Collect IDs of steps from the request
	requestedStepIDs := make([]int, 0)
	for _, step := range workflow.Steps {
		if step.ID != 0 {
			requestedStepIDs = append(requestedStepIDs, step.ID)
		}
	}

	// Delete removed steps
	err = DeleteRemovedSteps(ctx, workflow.Id, requestedStepIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to delete removed steps: %w", err)
	}

	// Update or insert steps
	for _, step := range workflow.Steps {
		payloadJSON, err := json.Marshal(step.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal step payload: %w", err)
		}

		if step.ID != 0 {
			// Update existing step
			updateStepQuery := `UPDATE steps SET name = $1, step_type = $2, payload = $3, step_order = $4 WHERE id = $5`
			_, err = ctx.SQL.ExecContext(ctx, updateStepQuery, step.Name, step.Type, string(payloadJSON), step.StepOrder, step.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to update step with ID %d: %w", step.ID, err)
			}
		} else {
			// Insert new step
			insertStepQuery := `INSERT INTO steps (workflow_id, name, step_type, payload, step_order) VALUES ($1, $2, $3, $4, $5)`
			_, err = ctx.SQL.ExecContext(ctx, insertStepQuery, workflow.Id, step.Name, step.Type, string(payloadJSON), step.StepOrder)
			if err != nil {
				return nil, fmt.Errorf("failed to insert new step: %w", err)
			}
		}
	}

	// Return the updated workflow
	return workflow, nil
}

func DeleteRemovedSteps(ctx *gofr.Context, workflowID int, stepIDs []int) error {
	if len(stepIDs) == 0 {
		// If no steps are specified, delete all steps for this workflow
		deleteQuery := `DELETE FROM steps WHERE workflow_id = $1`
		_, err := ctx.SQL.ExecContext(ctx, deleteQuery, workflowID)
		return err
	}

	// Convert the stepIDs slice into a comma-separated string
	stepIDStrings := make([]string, len(stepIDs))
	for i, id := range stepIDs {
		stepIDStrings[i] = fmt.Sprintf("%d", id)
	}
	idList := strings.Join(stepIDStrings, ",") // e.g., "1,2,3"

	// Use the constructed string in the SQL query
	deleteQuery := fmt.Sprintf(`DELETE FROM steps WHERE workflow_id = $1 AND id NOT IN (%s)`, idList)
	_, err := ctx.SQL.ExecContext(ctx, deleteQuery, workflowID)
	return err
}

// func UpdateWorkflow(ctx *gofr.Context) (interface{}, error) {
// 	var workflow Workflow

// 	// Bind the request body to the Workflow struct
// 	err := ctx.Bind(&workflow)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse workflow: %w", err)
// 	}

// 	// Update the workflow metadata (name and webhook URL)
// 	query := `UPDATE workflows SET name = $1, webhook_url = $2 WHERE id = $3`
// 	_, err = ctx.SQL.ExecContext(ctx, query, workflow.Name, workflow.WebookUrl, workflow.Id)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to update workflow: %w", err)
// 	}

// 	// Steps handling: update, insert, or delete steps
// 	for _, step := range workflow.Steps {
// 		if step.ID == 0 {
// 			// New step: insert into the database
// 			stepQuery := `INSERT INTO steps (workflow_id, name, step_type, payload, step_order)
//                           VALUES ($1, $2, $3, $4, $5)`
// 			payloadJSON, jsonErr := json.Marshal(step.Payload)
// 			if jsonErr != nil {
// 				return nil, fmt.Errorf("failed to serialize payload for step %s: %w", step.Name, jsonErr)
// 			}
// 			_, err := ctx.SQL.ExecContext(ctx, stepQuery, workflow.Id, step.Name, step.Type, payloadJSON, step.StepOrder)
// 			if err != nil {
// 				return nil, fmt.Errorf("failed to insert step %s: %w", step.Name, err)
// 			}
// 		} else {
// 			// Existing step: update in the database
// 			stepQuery := `UPDATE steps
//                           SET name = $1, step_type = $2, payload = $3, step_order = $4
//                           WHERE id = $5 AND workflow_id = $6`
// 			payloadJSON, jsonErr := json.Marshal(step.Payload)
// 			if jsonErr != nil {
// 				return nil, fmt.Errorf("failed to serialize payload for step %s: %w", step.Name, jsonErr)
// 			}
// 			_, err := ctx.SQL.ExecContext(ctx, stepQuery, step.Name, step.Type, payloadJSON, step.StepOrder, step.ID, workflow.Id)
// 			if err != nil {
// 				return nil, fmt.Errorf("failed to update step %s: %w", step.Name, err)
// 			}
// 		}
// 	}

// 	// Optionally: remove steps that are no longer present in the request
// 	// First, collect IDs of steps in the request
// 	requestedStepIDs := make(map[int]bool)
// 	for _, step := range workflow.Steps {
// 		if step.ID != 0 {
// 			requestedStepIDs[step.ID] = true
// 		}
// 	}

// 	// Find and delete steps not in the request
// 	deleteQuery := `DELETE FROM steps WHERE workflow_id = $1 AND id NOT IN (SELECT UNNEST($2::int[]))`
// 	var stepIDs []int
// 	for id := range requestedStepIDs {
// 		stepIDs = append(stepIDs, id)
// 	}
// 	_, err = ctx.SQL.ExecContext(ctx, deleteQuery, workflow.Id, stepIDs)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to delete removed steps: %w", err)
// 	}

// 	return workflow, nil
// }

func GetWorkflow(ctx *gofr.Context) (interface{}, error) {
	// Extract workflow ID from query parameters
	workflowID := ctx.Request.PathParam("id")
	if workflowID == "" {
		return nil, fmt.Errorf("workflow ID is required")
	}

	// Query to fetch the workflow details
	var workflow Workflow
	query := `SELECT id, name, webhook_url FROM workflows WHERE id = $1`
	err := ctx.SQL.QueryRowContext(ctx, query, workflowID).Scan(&workflow.Id, &workflow.Name, &workflow.WebookUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow: %w", err)
	}

	// Query to fetch the steps associated with the workflowpackage
	stepQuery := `SELECT id, name, step_type, payload, step_order FROM steps WHERE workflow_id = $1 ORDER BY step_order`
	rows, err := ctx.SQL.QueryContext(ctx, stepQuery, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch steps for workflow: %w", err)
	}
	defer rows.Close()

	// Parse the steps into the workflow object
	var steps []Step
	for rows.Next() {
		var step Step
		var payloadJSON string

		// Scan the row into the Step struct
		err := rows.Scan(&step.ID, &step.Name, &step.Type, &payloadJSON, &step.StepOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to parse step data: %w", err)
		}

		// Deserialize the JSON payload
		err = json.Unmarshal([]byte(payloadJSON), &step.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize step payload: %w", err)
		}

		steps = append(steps, step)
	}

	// Attach the steps to the workflow
	workflow.Steps = steps

	// Return the workflow
	return workflow, nil
}

// func webhookHandler(ctx *gofr.Context) (interface{}, error) {
// 	workflowID := ctx.Param("workflowId") // Extract the workflow ID from the URL
// 	var payload map[string]interface{}    // Generic map to hold the webhook payload

// 	err := ctx.Bind(&payload)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid webhook payload: %w", err)
// 	}

// 	// Fetch the workflow details
// 	var workflow Workflow
// 	query := `SELECT id, name, webhook_url FROM workflows WHERE id = $1`
// 	err = ctx.SQL.QueryRowContext(ctx, query, workflowID).Scan(&workflow.Id, &workflow.Name, &workflow.WebookUrl)
// 	if err != nil {
// 		return nil, fmt.Errorf("workflow not found: %w", err)
// 	}

// 	// Fetch steps for the workflow
// 	query = `SELECT id, name, step_type, payload, step_order FROM steps WHERE workflow_id = $1 ORDER BY step_order`
// 	rows, err := ctx.SQL.QueryContext(ctx, query, workflowID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch workflow steps: %w", err)
// 	}
// 	defer rows.Close()

// 	steps := []Step{}
// 	for rows.Next() {
// 		var step Step
// 		var payloadJSON string
// 		err = rows.Scan(&step.ID, &step.Name, &step.Type, &payloadJSON, &step.StepOrder)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to parse step data: %w", err)
// 		}

// 		// Decode the JSON payload into the step's Payload field
// 		err = json.Unmarshal([]byte(payloadJSON), &step.Payload)
// 		if err != nil {
// 			return nil, fmt.Errorf("invalid step payload: %w", err)
// 		}

// 		steps = append(steps, step)
// 	}

// 	workflow.Steps = steps

// 	// Execute the workflow
// 	err = executeWorkflow(ctx, workflow, payload)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to execute workflow: %w", err)
// 	}

// 	return "Workflow executed successfully", nil
// }

// func executeWorkflow(ctx *gofr.Context, workflow Workflow, input map[string]interface{}) error {
// 	var intermediateData map[string]interface{} = input

// 	for _, step := range workflow.Steps {
// 		var err error
// 		switch step.Type {
// 		case "trigger":
// 			intermediateData, err = executeTrigger(step, intermediateData)
// 		case "parse":
// 			intermediateData, err = executeParse(step, intermediateData)
// 		case "action":
// 			err = executeAction(step, intermediateData)
// 		default:
// 			return fmt.Errorf("unknown step type: %s", step.Type)
// 		}

// 		if err != nil {
// 			return fmt.Errorf("step '%s' failed: %w", step.Name, err)
// 		}
// 	}

// 	return nil
// }

// func executeTrigger(step Step, input map[string]interface{}) (map[string]interface{}, error) {
// 	// Process trigger logic
// 	return input, nil
// }

// func executeParse(step Step, input map[string]interface{}) (map[string]interface{}, error) {
// 	// Process parse logic (e.g., JSON to CSV conversion)
// 	// Use step.Payload for configurations like input/output formats
// 	return input, nil
// }

// func executeAction(step Step, input map[string]interface{}) error {
// 	// Perform external action (e.g., API call, email sending)
// 	// Use step.Payload for action-specific parameters
// 	return nil
// }

func webhookHandler(ctx *gofr.Context) (interface{}, error) {
	workflowID := ctx.Param("workflowId")
	var payload map[string]interface{}

	// Parse the incoming multipart form-data
	fileHeader, _, err := parseMultipartRequest(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipart request: %w", err)
	}

	// Save the uploaded file temporarily
	tempFilePath := "tmp/uploaded_file"
	if fileHeader != nil {
		err = saveUploadedFile(fileHeader, tempFilePath)
		if err != nil {
			return nil, fmt.Errorf("error saving uploaded file: %w", err)
		}
		defer os.Remove(tempFilePath) // Clean up after use
	}

	// Fetch workflow details
	var workflow Workflow
	query := `SELECT id, name, webhook_url FROM workflows WHERE id = $1`
	err = ctx.SQL.QueryRowContext(ctx, query, workflowID).Scan(&workflow.Id, &workflow.Name, &workflow.WebookUrl)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Fetch workflow steps
	query = `SELECT id, name, step_type, payload, step_order FROM steps WHERE workflow_id = $1 ORDER BY step_order`
	rows, err := ctx.SQL.QueryContext(ctx, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow steps: %w", err)
	}
	defer rows.Close()

	steps := []Step{}
	for rows.Next() {
		var step Step
		var payloadJSON string
		err = rows.Scan(&step.ID, &step.Name, &step.Type, &payloadJSON, &step.StepOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to parse step data: %w", err)
		}

		err = json.Unmarshal([]byte(payloadJSON), &step.Payload)
		if err != nil {
			return nil, fmt.Errorf("invalid step payload: %w", err)
		}

		steps = append(steps, step)
	}
	workflow.Steps = steps

	// Execute the workflow with the uploaded file or other payload data
	err = executeWorkflow(ctx, workflow, payload, tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to execute workflow: %w", err)
	}

	return "Workflow executed successfully", nil
}

// Parses a multipart request
func parseMultipartRequest(ctx *gofr.Context) (*multipart.FileHeader, map[string]interface{}, error) {
	var payload map[string]interface{}
	err := ctx.Bind(&payload)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid webhook payload: %w", err)
	}

	fileHeader := ctx.FileHeader("file")
	return fileHeader, payload, nil
}

// Saves an uploaded file to the local filesystem
func saveUploadedFile(fileHeader *multipart.FileHeader, filePath string) error {
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("error opening uploaded file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading uploaded file content: %w", err)
	}

	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		return fmt.Errorf("error saving uploaded file: %w", err)
	}
	return nil
}

// Executes the workflow with the provided input data and steps
func executeWorkflow(ctx *gofr.Context, workflow Workflow, input map[string]interface{}, tempFilePath string) error {
	intermediateData := input

	for _, step := range workflow.Steps {
		var err error
		switch step.Type {
		case "trigger":
			intermediateData, err = executeTrigger(step, intermediateData)
		case "parse":
			intermediateData, err = executeParse(step, intermediateData, tempFilePath)
		case "action":
			err = executeAction(step, intermediateData)
		default:
			return fmt.Errorf("unknown step type: %s", step.Type)
		}

		if err != nil {
			return fmt.Errorf("step '%s' failed: %w", step.Name, err)
		}
	}

	return nil
}

// Trigger step execution
func executeTrigger(step Step, input map[string]interface{}) (map[string]interface{}, error) {
	// Process trigger logic
	return input, nil
}

// Parse step execution
func executeParse(step Step, input map[string]interface{}, filePath string) (map[string]interface{}, error) {
	// Example: Convert JSON to CSV, or process file-specific logic
	if _, ok := step.Payload["format"].(string); ok {
		fmt.Printf("Parsing file %s with format %s\n", filePath, step.Payload["format"].(string))
	}
	return input, nil
}

// Action step execution
func executeAction(step Step, input map[string]interface{}) error {
	// Example: Make API calls or perform specific actions
	fmt.Printf("Performing action '%s' with payload %v\n", step.Name, step.Payload)
	return nil
}
