-- Create workflow_executions table to track scheduled workflow runs
CREATE TABLE IF NOT EXISTS workflow_executions (
    id SERIAL PRIMARY KEY,
    workflow_id INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    message TEXT,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    
    CONSTRAINT fk_workflow_executions_workflow
        FOREIGN KEY (workflow_id) 
        REFERENCES workflows (id) 
        ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_workflow_executions_workflow_id 
    ON workflow_executions (workflow_id);

CREATE INDEX IF NOT EXISTS idx_workflow_executions_executed_at 
    ON workflow_executions (executed_at);

CREATE INDEX IF NOT EXISTS idx_workflow_executions_status 
    ON workflow_executions (status);

-- Add column to workflows table to track if it should run on schedule
ALTER TABLE workflows 
ADD COLUMN IF NOT EXISTS active BOOLEAN DEFAULT true;

-- Comment on table
COMMENT ON TABLE workflow_executions IS 'Tracks execution history of scheduled workflows';
