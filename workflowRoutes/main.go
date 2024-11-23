package workflowRoutes

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"

	"gofr.dev/pkg/gofr"
)

type Workflow struct {
	WebookUrl string `json:"webhookUrl"`
	Id        int    `json:"id"`
	Steps     []Step `json:"steps"`
	Name      string `json:"name"`
	Uid       User
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
	err := ctx.Bind(&workflow)
	if err != nil {
		return nil, err
	}
	webhookUrl, webhookUrlErr := GenerateWebhookUrl()
	if webhookUrlErr != nil {
		return nil, webhookUrlErr
	}

	query := `INSERT INTO workflows (name, webhook_url) VALUES ($1, $2) RETURNING id`
	err = ctx.SQL.QueryRowContext(ctx, query, workflow.Name, webhookUrl).Scan(&workflow.Id)
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

// func getWorkflow(ctx *gofr.Context) interface{} {
// 	// userId := ctx.Value("uid")
// 	workflowId := ctx.Param("id")
// 	workflowQuery := `SELECT * FROM workflows WHERE id = $1`
// 	// var workflow Workflow
// 	rows, qerr := ctx.SQL.QueryContext(ctx, workflowQuery, workflowId)
// 	if qerr != nil {
// 		return nil, qerr
// 	}
// var workflows []Workflow
// // for rows.Next()
// }
