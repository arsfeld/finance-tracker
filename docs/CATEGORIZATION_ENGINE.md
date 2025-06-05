# Transaction Categorization Engine

## Overview

The Finaro Transaction Categorization Engine is a sophisticated multi-layer system that automatically categorizes financial transactions using a combination of rule-based matching, pattern recognition, batch LLM processing, and RAG-enhanced learning. The engine is designed to be cost-efficient, accurate, and continuously improving through user feedback.

## Architecture

### Core Philosophy

1. **Cost Efficiency**: Minimize LLM API costs through intelligent batching and confidence-based processing
2. **Accuracy**: Multi-layer approach ensures high categorization accuracy
3. **Learning**: RAG system learns from user corrections and improves over time
4. **Performance**: Optimized for real-time categorization during transaction sync
5. **Multi-Tenancy**: Organization-specific learning while maintaining privacy

### Processing Pipeline

```
Incoming Transactions
    ↓
Layer 1: Rule-Based Engine (Instant, Free)
    ↓ (if confidence < 90%)
Layer 2: Pattern Matching Engine (Fast, Free)
    ↓ (if confidence < 80%)
Layer 3: RAG Similarity Search (Medium, Free)
    ↓ (if confidence < 70%)
Layer 4: Batch LLM Processing (Slow, Paid)
    ↓
Confidence Scoring & Final Assignment
    ↓
User Feedback Collection & Learning
```

## Engine Components

### 1. Rule-Based Engine

**Purpose**: Fast, deterministic categorization using predefined rules
**Cost**: Free
**Accuracy**: 95%+ when rules match

```go
type RuleEngine struct {
    rules       []CategoryRule
    patterns    []MerchantPattern
    keywords    []KeywordRule
    amounts     []AmountRule
}

type CategoryRule struct {
    ID            int      `json:"id"`
    CategoryID    int      `json:"category_id"`
    Field         string   `json:"field"`         // "description", "merchant_name", "amount"
    Operator      string   `json:"operator"`      // "contains", "equals", "regex", "amount_range"
    Value         string   `json:"value"`
    CaseSensitive bool     `json:"case_sensitive"`
    Priority      int      `json:"priority"`      // Higher number = higher priority
    Confidence    float64  `json:"confidence"`    // 0.0-1.0
}
```

### 2. Pattern Matching Engine

**Purpose**: Merchant and description pattern recognition
**Cost**: Free
**Accuracy**: 85%+ for known patterns

```go
type PatternEngine struct {
    merchantPatterns    map[string]CategoryMatch
    descriptionPatterns map[string]CategoryMatch
    amountRanges       []AmountPattern
    fuzzyMatcher       FuzzyMatcher
}

type CategoryMatch struct {
    CategoryID   int     `json:"category_id"`
    Confidence   float64 `json:"confidence"`
    MatchType    string  `json:"match_type"`  // "exact", "fuzzy", "partial"
    LastUsed     time.Time `json:"last_used"`
    UsageCount   int     `json:"usage_count"`
}

type MerchantPattern struct {
    Pattern     string  `json:"pattern"`      // "STARBUCKS*", "AMAZON.COM*"
    CategoryID  int     `json:"category_id"`
    Confidence  float64 `json:"confidence"`
    IsRegex     bool    `json:"is_regex"`
}
```

### 3. RAG Similarity Engine

**Purpose**: Learn from similar transactions and user feedback
**Cost**: Free (after initial embedding creation)
**Accuracy**: 80%+ for similar transaction patterns

```go
type RAGEngine struct {
    vectorDB     VectorDatabase
    embeddings   EmbeddingService
    similarity   SimilarityService
    feedback     FeedbackStore
    context      OrganizationContext
}

type TransactionEmbedding struct {
    TransactionID uuid.UUID `json:"transaction_id"`
    Embedding     []float64 `json:"embedding"`      // 1536-dim vector
    CategoryID    int       `json:"category_id"`
    Confidence    float64   `json:"confidence"`
    CreatedAt     time.Time `json:"created_at"`
}

type SimilarityMatch struct {
    TransactionID   uuid.UUID `json:"transaction_id"`
    CategoryID      int       `json:"category_id"`
    Similarity      float64   `json:"similarity"`     // Cosine similarity 0.0-1.0
    Description     string    `json:"description"`
    MerchantName    string    `json:"merchant_name"`
    Amount          float64   `json:"amount"`
}
```

### 4. Batch LLM Engine

**Purpose**: AI-powered categorization for complex/unknown transactions
**Cost**: Variable ($0.001-0.01 per transaction depending on model)
**Accuracy**: 90%+ for complex categorizations

```go
type LLMCategorizationEngine struct {
    client       LLMClient
    batchConfig  BatchConfig
    costTracker  *CostTracker
    rateLimiter  *RateLimiter
    models       []LLMModel
}

type BatchConfig struct {
    MaxBatchSize     int           `json:"max_batch_size"`     // 100
    MinBatchSize     int           `json:"min_batch_size"`     // 20
    TimeWindow       time.Duration `json:"time_window"`        // 1 hour
    ConfidenceThresh float64       `json:"confidence_thresh"`  // 0.7
}

type LLMModel struct {
    Name        string  `json:"name"`         // "gpt-4o-mini", "claude-3-haiku"
    CostPer1K   float64 `json:"cost_per_1k"`  // Cost per 1K tokens
    MaxTokens   int     `json:"max_tokens"`   // Model context limit
    Accuracy    float64 `json:"accuracy"`     // Historical accuracy score
    IsDefault   bool    `json:"is_default"`
}
```

## Database Schema

### Extended Transaction Metadata

```sql
-- Extend existing transactions table
ALTER TABLE transactions 
ADD COLUMN IF NOT EXISTS categorization_metadata JSONB DEFAULT '{
    "confidence_score": null,
    "categorization_method": null,
    "llm_batch_id": null,
    "user_corrected": false,
    "similarity_matches": [],
    "rule_matches": [],
    "processing_time_ms": null,
    "cost_estimate": null
}'::jsonb;

-- Performance indexes
CREATE INDEX idx_transactions_categorization ON transactions USING GIN (categorization_metadata);
CREATE INDEX idx_transactions_confidence ON transactions ((categorization_metadata->>'confidence_score')::numeric);
CREATE INDEX idx_transactions_method ON transactions ((categorization_metadata->>'categorization_method'));
CREATE INDEX idx_transactions_uncategorized ON transactions (category_id) WHERE category_id IS NULL;
```

### New Supporting Tables

```sql
-- Transaction embeddings for RAG
CREATE TABLE transaction_embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    embedding vector(1536),  -- OpenAI embedding dimension
    embedding_model TEXT NOT NULL DEFAULT 'text-embedding-3-small',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(transaction_id)
);

-- Vector similarity index
CREATE INDEX idx_transaction_embeddings_vector ON transaction_embeddings USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_transaction_embeddings_org ON transaction_embeddings (organization_id);

-- LLM batch tracking for cost optimization
CREATE TABLE llm_categorization_batches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    transaction_count INTEGER NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    total_cost DECIMAL(10,6) NOT NULL,
    model_used TEXT NOT NULL,
    batch_type TEXT NOT NULL DEFAULT 'categorization',
    success_rate DECIMAL(5,4), -- Percentage of successful categorizations
    avg_confidence DECIMAL(5,4), -- Average confidence score
    processing_time_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Cost tracking index
CREATE INDEX idx_llm_batches_org_date ON llm_categorization_batches (organization_id, created_at);
CREATE INDEX idx_llm_batches_cost ON llm_categorization_batches (total_cost, created_at);

-- User feedback for learning
CREATE TABLE categorization_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id),
    old_category_id INTEGER REFERENCES categories(id),
    new_category_id INTEGER NOT NULL REFERENCES categories(id),
    feedback_type TEXT NOT NULL CHECK (feedback_type IN ('correction', 'confirmation', 'rejection')),
    confidence_before DECIMAL(5,4),
    method_used TEXT, -- Which categorization method was corrected
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Feedback analysis indexes
CREATE INDEX idx_feedback_org_user ON categorization_feedback (organization_id, user_id);
CREATE INDEX idx_feedback_transaction ON categorization_feedback (transaction_id);
CREATE INDEX idx_feedback_categories ON categorization_feedback (old_category_id, new_category_id);

-- Enhanced category rules
CREATE TABLE category_rules_enhanced (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    rule_type TEXT NOT NULL CHECK (rule_type IN ('merchant_pattern', 'description_keyword', 'amount_range', 'regex_pattern')),
    pattern TEXT NOT NULL,
    confidence DECIMAL(5,4) NOT NULL DEFAULT 0.9,
    priority INTEGER NOT NULL DEFAULT 100,
    is_case_sensitive BOOLEAN NOT NULL DEFAULT false,
    is_regex BOOLEAN NOT NULL DEFAULT false,
    usage_count INTEGER NOT NULL DEFAULT 0,
    success_rate DECIMAL(5,4) DEFAULT 1.0,
    created_by UUID REFERENCES auth.users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

-- Rule performance indexes
CREATE INDEX idx_category_rules_org ON category_rules_enhanced (organization_id);
CREATE INDEX idx_category_rules_type ON category_rules_enhanced (rule_type);
CREATE INDEX idx_category_rules_priority ON category_rules_enhanced (priority DESC);
CREATE INDEX idx_category_rules_pattern ON category_rules_enhanced USING gin (pattern gin_trgm_ops);

-- Pattern matching cache for performance
CREATE TABLE merchant_patterns_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    merchant_pattern TEXT NOT NULL,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    confidence DECIMAL(5,4) NOT NULL,
    usage_count INTEGER NOT NULL DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, merchant_pattern)
);

-- Pattern cache indexes
CREATE INDEX idx_merchant_cache_org ON merchant_patterns_cache (organization_id);
CREATE INDEX idx_merchant_cache_pattern ON merchant_patterns_cache USING gin (merchant_pattern gin_trgm_ops);
CREATE INDEX idx_merchant_cache_usage ON merchant_patterns_cache (usage_count DESC, last_used_at DESC);
```

## River Job Integration

### Job Types

```go
// Batch categorization job
type BatchCategorizationJob struct {
    OrganizationID    uuid.UUID   `json:"organization_id"`
    TransactionIDs    []uuid.UUID `json:"transaction_ids,omitempty"` // Specific transactions
    DateRange         *DateRange  `json:"date_range,omitempty"`      // Or date range
    ForceRecategorize bool        `json:"force_recategorize"`        // Recategorize existing
    MaxCost           float64     `json:"max_cost"`                  // Budget limit
}

// RAG learning job
type RAGLearningJob struct {
    OrganizationID uuid.UUID   `json:"organization_id"`
    FeedbackIDs    []uuid.UUID `json:"feedback_ids,omitempty"`
    RebuildIndex   bool        `json:"rebuild_index"`
}

// Pattern mining job
type PatternMiningJob struct {
    OrganizationID uuid.UUID `json:"organization_id"`
    MinConfidence  float64   `json:"min_confidence"`
    MinUsageCount  int       `json:"min_usage_count"`
}

// Real-time categorization job (triggered on transaction sync)
type RealtimeCategorizationJob struct {
    TransactionID  uuid.UUID `json:"transaction_id"`
    OrganizationID uuid.UUID `json:"organization_id"`
    Priority       string    `json:"priority"` // "high", "normal", "low"
}
```

### Queue Configuration

```go
// Queue: categorization (5 workers)
// - Real-time categorization during sync
// - High priority for user-facing operations

// Queue: batch_categorization (3 workers)  
// - Batch LLM processing
// - Cost-optimized processing

// Queue: rag_learning (2 workers)
// - Pattern learning from feedback
// - Embedding generation and updates
```

## Cost Management

### Budget Controls

```go
type CostManager struct {
    monthlyBudget   float64
    dailyBudget     float64
    currentSpend    float64
    tokenCount      int64
    avgCostPerTxn   float64
    costAlerts      []CostAlert
}

type CostAlert struct {
    Threshold  float64 `json:"threshold"`   // 0.8 = 80% of budget
    AlertType  string  `json:"alert_type"` // "email", "slack", "disable"
    Recipients []string `json:"recipients"`
}
```

### Model Selection Strategy

```go
type ModelSelector struct {
    models []LLMModel
    strategy string // "cost_optimized", "accuracy_optimized", "balanced"
}

// Cost-optimized: GPT-4o-mini -> Claude Haiku -> Disable
// Accuracy-optimized: GPT-4 -> Claude Sonnet -> GPT-4o-mini  
// Balanced: GPT-4o-mini -> Claude Sonnet -> Disable
```

## Performance Targets

### Categorization Speed
- **Rule-based**: < 1ms per transaction
- **Pattern matching**: < 5ms per transaction  
- **RAG similarity**: < 50ms per transaction
- **Batch LLM**: < 2 seconds per batch (50-100 transactions)

### Accuracy Targets
- **Overall accuracy**: > 92%
- **Rule-based**: > 95%
- **Pattern matching**: > 85%
- **RAG similarity**: > 80%
- **LLM categorization**: > 90%

### Cost Targets
- **Average cost per transaction**: < $0.003
- **Monthly cost per organization**: < $10
- **LLM usage**: < 20% of transactions

## Monitoring & Analytics

### Key Metrics
- Categorization accuracy by method
- Cost per transaction and per organization
- Processing time by engine layer
- User correction rates
- Model performance comparison
- Pattern learning effectiveness

### Dashboards
- Real-time categorization performance
- Cost tracking and budget alerts
- Accuracy trends over time
- User feedback analysis
- Engine layer utilization

## Implementation Phases

### Phase 1: Foundation (Week 1)
- [ ] Database schema migration
- [ ] Enhanced rule engine implementation
- [ ] Basic pattern matching
- [ ] Confidence scoring framework

### Phase 2: LLM Integration (Week 2)  
- [ ] Batch LLM categorization
- [ ] Cost tracking and budget controls
- [ ] River job integration
- [ ] Model selection logic

### Phase 3: RAG Learning (Week 3)
- [ ] Vector database integration
- [ ] Embedding generation
- [ ] Similarity search implementation
- [ ] User feedback collection

### Phase 4: Optimization (Week 4)
- [ ] Performance tuning
- [ ] Advanced pattern mining
- [ ] Monitoring and analytics
- [ ] Production deployment

## Configuration

### Environment Variables
```bash
# LLM Configuration
CATEGORIZATION_LLM_PROVIDER=openrouter
CATEGORIZATION_LLM_MODELS=gpt-4o-mini,claude-3-haiku
CATEGORIZATION_MONTHLY_BUDGET=50.00
CATEGORIZATION_BATCH_SIZE=75
CATEGORIZATION_CONFIDENCE_THRESHOLD=0.7

# Vector Database (if using external service)
CATEGORIZATION_VECTOR_DB_URL=postgresql://...
CATEGORIZATION_EMBEDDING_MODEL=text-embedding-3-small

# Performance Tuning
CATEGORIZATION_CACHE_TTL=3600
CATEGORIZATION_MAX_CONCURRENT_BATCHES=3
CATEGORIZATION_PATTERN_CACHE_SIZE=10000
```

### Organization Settings
```json
{
  "categorization": {
    "enabled": true,
    "auto_categorize": true,
    "confidence_threshold": 0.7,
    "max_monthly_cost": 25.0,
    "preferred_models": ["gpt-4o-mini"],
    "learning_enabled": true,
    "batch_processing": true
  }
}
```

This comprehensive documentation provides the foundation for implementing a sophisticated, cost-efficient transaction categorization engine that will significantly improve user experience while maintaining budget control.