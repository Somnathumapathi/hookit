package main

import (
	"context"
	"github/Somnathumapathi/gofrhack/authRoutes"
	"github/Somnathumapathi/gofrhack/cmRoutes"
	"github/Somnathumapathi/gofrhack/cronRoutes"
	"github/Somnathumapathi/gofrhack/services"
	"github/Somnathumapathi/gofrhack/testRoutes"
	"github/Somnathumapathi/gofrhack/workflowRoutes"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"gofr.dev/pkg/gofr"
)

func baseHandler(ctx *gofr.Context) (interface{}, error) {
	resp := make(map[string]string)
	resp["message"] = "Server up!"
	return resp, nil
}

var jwtKey = []byte("my_secret_key")

// Claims : type for jwt body
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func authMiddleware() func(handler http.Handler) http.Handler {
	return func(inner http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")
			// if request URI is /users/create or /users/login or /(base url) then no need for authentication check
			requestURI := r.RequestURI
			if requestURI == "/user/create" || requestURI == "/user/login" || requestURI == "/" {
				//no need for authentication
				// sends the request to the next middleware/request-handler
				inner.ServeHTTP(w, r)
			} else {
				//user has to be authenticated here
				authHeader := r.Header["Authorization"]
				if authHeader == nil {
					http.Error(w, "Not authorized. Please login!", http.StatusUnauthorized)
					return
				}
				if authHeader[0] == "" {
					http.Error(w, "Not authorized. Please login!", http.StatusUnauthorized)
					return
				}
				tokenStr := strings.Split(authHeader[0], " ")
				if len(tokenStr) == 1 {
					http.Error(w, "Not authorized. Please login!", http.StatusUnauthorized)
					return
				}

				claims := &Claims{}

				// Parse the JWT string and store the result in `claims`.
				// Note that we are passing the key in this method as well. This method will return an error
				// if the token is invalid (if it has expired according to the expiry time we set on sign in),
				// or if the signature does not match
				tkn, err := jwt.ParseWithClaims(tokenStr[1], claims, func(token *jwt.Token) (any, error) {
					return jwtKey, nil
				})
				if err != nil || !tkn.Valid {
					http.Error(w, "You are unauthorized. Please login again!", http.StatusUnauthorized)
					return
				}

				//token is valid and not expired hence a valid and authorized user
				newContext := context.WithValue(r.Context(), "userEmail", claims.Username)
				newR := r.Clone(newContext)

				// sends the request to the next middleware/request-handler
				inner.ServeHTTP(w, newR)
			}
		})
	}
}

func main() {
	// initialise gofr object
	app := gofr.New()

	// Initialize and start cron service for scheduled workflows
	cronService := services.NewCronService(app)

	// Start scheduled workflows when the server starts
	// We'll do this after the server is running to ensure database connections are ready
	go func() {
		// Wait a moment for the server to fully initialize
		// In a production setup, you'd want to use proper synchronization
		// time.Sleep(2 * time.Second)

		// Use a simple handler context to start cron jobs
		// This is a workaround since we need a context with SQL access
		app.GET("/internal/start-cron", func(ctx *gofr.Context) (interface{}, error) {
			err := cronService.StartScheduledWorkflows(ctx)
			if err != nil {
				ctx.Logger.Errorf("Failed to start scheduled workflows: %v", err)
				return map[string]string{"error": err.Error()}, nil
			}
			ctx.Logger.Info("Scheduled workflows service started successfully")
			return map[string]string{"message": "Cron service started"}, nil
		})
	}()

	// Public routes (no auth required)
	app.GET("/", func(ctx *gofr.Context) (interface{}, error) {
		return map[string]string{"message": "Hookit API Server is running!", "version": "1.0.0"}, nil
	})

	// Auth routes
	app.POST("/user/register", authRoutes.RegisterUser)
	app.POST("/user/login", authRoutes.LoginUser)

	// Workflow routes (with auth middleware would be added here)
	app.POST("/workflow/create", workflowRoutes.CreateWorkflow)
	app.GET("/workflow/{id}", workflowRoutes.GetWorkflow)
	app.GET("/workflows/{uid}", workflowRoutes.GetWorkflows) // List all workflows for a user
	app.PUT("/workflow/{id}", workflowRoutes.UpdateWorkflow)

	// Cron/Schedule management routes
	app.GET("/scheduled-workflows", cronRoutes.GetScheduledWorkflows)
	app.GET("/workflow/{workflowId}/executions", cronRoutes.GetWorkflowExecutions)
	app.PUT("/workflow/{workflowId}/schedule", cronRoutes.ToggleWorkflowSchedule)

	// Credit management
	app.POST("/buyCredits", cmRoutes.AddCreditsHandler)
	app.GET("/user/{userId}/credits", cmRoutes.GetUserCredits)

	// Test/Demo routes for cron functionality
	app.POST("/test/create-scheduled-workflow", testRoutes.CreateTestScheduledWorkflow)
	app.POST("/test/execute/{workflowId}", testRoutes.TestCronExecution)
	app.GET("/test/cron-status", testRoutes.GetCronStatus)

	// Webhook execution endpoint
	app.POST("/webhook/{workflowId}", workflowRoutes.ExecuteWorkflow)

	app.Run()
}
