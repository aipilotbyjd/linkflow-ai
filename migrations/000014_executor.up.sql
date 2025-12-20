-- ============================================================================
-- Migration: 000014_executor
-- Description: Executor workers and tasks
-- ============================================================================

CREATE TABLE executor_workers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255),
    port INTEGER,
    status VARCHAR(50) NOT NULL DEFAULT 'idle',
    capabilities JSONB DEFAULT '[]',
    current_load INTEGER DEFAULT 0,
    max_load INTEGER DEFAULT 10,
    last_heartbeat_at TIMESTAMPTZ,
    version VARCHAR(50),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_executor_workers_status ON executor_workers(status);
CREATE INDEX idx_executor_workers_last_heartbeat ON executor_workers(last_heartbeat_at);

CREATE TABLE execution_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    node_id VARCHAR(255) NOT NULL,
    worker_id UUID REFERENCES executor_workers(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority INTEGER DEFAULT 0,
    input_data JSONB,
    output_data JSONB,
    error TEXT,
    error_details JSONB,
    timeout_seconds INTEGER DEFAULT 300,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    scheduled_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_execution_tasks_execution_id ON execution_tasks(execution_id);
CREATE INDEX idx_execution_tasks_worker_id ON execution_tasks(worker_id);
CREATE INDEX idx_execution_tasks_status ON execution_tasks(status);
CREATE INDEX idx_execution_tasks_priority ON execution_tasks(priority DESC, created_at ASC);
CREATE INDEX idx_execution_tasks_scheduled_at ON execution_tasks(scheduled_at) WHERE status = 'pending';

CREATE TABLE task_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    priority INTEGER DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_queue_status ON task_queue(status);
CREATE INDEX idx_task_queue_priority ON task_queue(priority DESC, scheduled_at ASC);
CREATE INDEX idx_task_queue_task_type ON task_queue(task_type);
