
-- ResolveFlow AI: Initial database schema

-- 1. Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role VARCHAR(30) NOT NULL DEFAULT 'agent'
        CHECK (role IN ('admin', 'manager', 'agent')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Support teams
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Team memberships
CREATE TABLE team_members (
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_role VARCHAR(30) NOT NULL DEFAULT 'member'
        CHECK (team_role IN ('lead', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (team_id, user_id)
);

-- 4. Customer support tickets
CREATE TABLE tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_number BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    subject VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    customer_name VARCHAR(150) NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'open'
        CHECK (status IN (
            'open', 'in_progress', 'waiting_for_customer',
            'resolved', 'closed'
        )),
    priority VARCHAR(20) NOT NULL DEFAULT 'medium'
        CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    channel VARCHAR(20) NOT NULL DEFAULT 'web'
        CHECK (channel IN ('web', 'email', 'chat', 'phone')),
    category VARCHAR(100),
    assigned_team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- 5. Ticket comments and replies
CREATE TABLE ticket_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    is_internal BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Ticket activity and audit history
CREATE TABLE ticket_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 7. AI analysis and predictions
CREATE TABLE ai_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    model_version VARCHAR(100) NOT NULL,
    category VARCHAR(100),
    predicted_priority VARCHAR(20)
        CHECK (
            predicted_priority IS NULL OR
            predicted_priority IN ('low', 'medium', 'high', 'urgent')
        ),
    sentiment VARCHAR(20)
        CHECK (
            sentiment IS NULL OR
            sentiment IN ('positive', 'neutral', 'negative')
        ),
    confidence NUMERIC(5, 4)
        CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
    suggested_team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    response_suggestion TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_tickets_status ON tickets(status);
CREATE INDEX idx_tickets_priority ON tickets(priority);
CREATE INDEX idx_tickets_assigned_to ON tickets(assigned_to);
CREATE INDEX idx_tickets_assigned_team ON tickets(assigned_team_id);
CREATE INDEX idx_tickets_created_at ON tickets(created_at DESC);

CREATE INDEX idx_ticket_comments_ticket
    ON ticket_comments(ticket_id, created_at);

CREATE INDEX idx_ticket_history_ticket
    ON ticket_history(ticket_id, created_at DESC);

CREATE INDEX idx_ai_predictions_ticket
    ON ai_predictions(ticket_id, created_at DESC);

CREATE INDEX idx_users_email ON users(email);
