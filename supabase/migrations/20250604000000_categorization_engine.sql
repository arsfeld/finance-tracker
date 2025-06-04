-- WalletMind Categorization Engine Migration
-- This migration adds the enhanced categorization system with multi-layer processing

-- Enable pg_trgm extension for fuzzy text matching
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Enable vector extension for embeddings (if available)
-- CREATE EXTENSION IF NOT EXISTS vector;

-- Extend transactions table with categorization metadata
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

-- Performance indexes for categorization metadata
CREATE INDEX IF NOT EXISTS idx_transactions_categorization ON transactions USING GIN (categorization_metadata);
CREATE INDEX IF NOT EXISTS idx_transactions_confidence ON transactions ((categorization_metadata->>'confidence_score')::numeric);
CREATE INDEX IF NOT EXISTS idx_transactions_method ON transactions ((categorization_metadata->>'categorization_method'));
CREATE INDEX IF NOT EXISTS idx_transactions_uncategorized ON transactions (category_id) WHERE category_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_transactions_merchant_search ON transactions USING gin (merchant_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_transactions_description_search ON transactions USING gin (description gin_trgm_ops);

-- Transaction embeddings for RAG (when vector extension is available)
-- Commented out until vector extension is confirmed available
/*
CREATE TABLE IF NOT EXISTS transaction_embeddings (
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
CREATE INDEX IF NOT EXISTS idx_transaction_embeddings_vector ON transaction_embeddings USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_transaction_embeddings_org ON transaction_embeddings (organization_id);
*/

-- LLM batch tracking for cost optimization
CREATE TABLE IF NOT EXISTS llm_categorization_batches (
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

-- Cost tracking indexes
CREATE INDEX IF NOT EXISTS idx_llm_batches_org_date ON llm_categorization_batches (organization_id, created_at);
CREATE INDEX IF NOT EXISTS idx_llm_batches_cost ON llm_categorization_batches (total_cost, created_at);

-- User feedback for learning
CREATE TABLE IF NOT EXISTS categorization_feedback (
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
CREATE INDEX IF NOT EXISTS idx_feedback_org_user ON categorization_feedback (organization_id, user_id);
CREATE INDEX IF NOT EXISTS idx_feedback_transaction ON categorization_feedback (transaction_id);
CREATE INDEX IF NOT EXISTS idx_feedback_categories ON categorization_feedback (old_category_id, new_category_id);

-- Enhanced category rules
CREATE TABLE IF NOT EXISTS category_rules_enhanced (
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
CREATE INDEX IF NOT EXISTS idx_category_rules_org ON category_rules_enhanced (organization_id);
CREATE INDEX IF NOT EXISTS idx_category_rules_type ON category_rules_enhanced (rule_type);
CREATE INDEX IF NOT EXISTS idx_category_rules_priority ON category_rules_enhanced (priority DESC);
CREATE INDEX IF NOT EXISTS idx_category_rules_pattern ON category_rules_enhanced USING gin (pattern gin_trgm_ops);

-- Pattern matching cache for performance
CREATE TABLE IF NOT EXISTS merchant_patterns_cache (
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
CREATE INDEX IF NOT EXISTS idx_merchant_cache_org ON merchant_patterns_cache (organization_id);
CREATE INDEX IF NOT EXISTS idx_merchant_cache_pattern ON merchant_patterns_cache USING gin (merchant_pattern gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_merchant_cache_usage ON merchant_patterns_cache (usage_count DESC, last_used_at DESC);

-- Enable RLS on new tables
ALTER TABLE llm_categorization_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE categorization_feedback ENABLE ROW LEVEL SECURITY;
ALTER TABLE category_rules_enhanced ENABLE ROW LEVEL SECURITY;
ALTER TABLE merchant_patterns_cache ENABLE ROW LEVEL SECURITY;

-- RLS policies for llm_categorization_batches
CREATE POLICY "Members can view llm batches" ON llm_categorization_batches
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = llm_categorization_batches.organization_id
            AND organization_members.user_id = auth.uid()
        )
    );

-- RLS policies for categorization_feedback
CREATE POLICY "Members can view feedback" ON categorization_feedback
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = categorization_feedback.organization_id
            AND organization_members.user_id = auth.uid()
        )
    );

CREATE POLICY "Members can manage feedback" ON categorization_feedback
    FOR ALL USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = categorization_feedback.organization_id
            AND organization_members.user_id = auth.uid()
            AND organization_members.role IN ('owner', 'admin', 'member')
        )
    );

-- RLS policies for category_rules_enhanced
CREATE POLICY "Members can view rules" ON category_rules_enhanced
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = category_rules_enhanced.organization_id
            AND organization_members.user_id = auth.uid()
        )
    );

CREATE POLICY "Members can manage rules" ON category_rules_enhanced
    FOR ALL USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = category_rules_enhanced.organization_id
            AND organization_members.user_id = auth.uid()
            AND organization_members.role IN ('owner', 'admin', 'member')
        )
    );

-- RLS policies for merchant_patterns_cache
CREATE POLICY "Members can view patterns" ON merchant_patterns_cache
    FOR SELECT USING (
        EXISTS (
            SELECT 1 FROM organization_members
            WHERE organization_members.organization_id = merchant_patterns_cache.organization_id
            AND organization_members.user_id = auth.uid()
        )
    );

-- Function to update pattern cache usage
CREATE OR REPLACE FUNCTION update_pattern_usage(
    p_organization_id UUID,
    p_merchant_pattern TEXT,
    p_category_id INTEGER,
    p_confidence DECIMAL(5,4)
) RETURNS void AS $$
BEGIN
    INSERT INTO merchant_patterns_cache (
        organization_id, 
        merchant_pattern, 
        category_id, 
        confidence, 
        usage_count, 
        last_used_at
    ) VALUES (
        p_organization_id,
        p_merchant_pattern,
        p_category_id,
        p_confidence,
        1,
        NOW()
    )
    ON CONFLICT (organization_id, merchant_pattern)
    DO UPDATE SET
        usage_count = merchant_patterns_cache.usage_count + 1,
        last_used_at = NOW(),
        confidence = GREATEST(merchant_patterns_cache.confidence, p_confidence);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to get similar merchant patterns
CREATE OR REPLACE FUNCTION get_similar_merchant_patterns(
    p_organization_id UUID,
    p_merchant_name TEXT,
    p_similarity_threshold DECIMAL DEFAULT 0.3,
    p_limit INTEGER DEFAULT 10
) RETURNS TABLE (
    merchant_pattern TEXT,
    category_id INTEGER,
    confidence DECIMAL(5,4),
    similarity DECIMAL(5,4),
    usage_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        mpc.merchant_pattern,
        mpc.category_id,
        mpc.confidence,
        similarity(mpc.merchant_pattern, p_merchant_name) as similarity,
        mpc.usage_count
    FROM merchant_patterns_cache mpc
    WHERE mpc.organization_id = p_organization_id
        AND similarity(mpc.merchant_pattern, p_merchant_name) > p_similarity_threshold
    ORDER BY 
        similarity DESC,
        mpc.usage_count DESC,
        mpc.last_used_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to record categorization feedback
CREATE OR REPLACE FUNCTION record_categorization_feedback(
    p_transaction_id UUID,
    p_old_category_id INTEGER,
    p_new_category_id INTEGER,
    p_feedback_type TEXT,
    p_confidence_before DECIMAL(5,4) DEFAULT NULL,
    p_method_used TEXT DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    feedback_id UUID;
    org_id UUID;
BEGIN
    -- Get organization_id from transaction
    SELECT organization_id INTO org_id 
    FROM transactions 
    WHERE id = p_transaction_id;
    
    IF org_id IS NULL THEN
        RAISE EXCEPTION 'Transaction not found';
    END IF;
    
    -- Insert feedback record
    INSERT INTO categorization_feedback (
        transaction_id,
        organization_id,
        user_id,
        old_category_id,
        new_category_id,
        feedback_type,
        confidence_before,
        method_used
    ) VALUES (
        p_transaction_id,
        org_id,
        auth.uid(),
        p_old_category_id,
        p_new_category_id,
        p_feedback_type,
        p_confidence_before,
        p_method_used
    ) RETURNING id INTO feedback_id;
    
    -- Update transaction categorization metadata
    UPDATE transactions 
    SET categorization_metadata = categorization_metadata || jsonb_build_object(
        'user_corrected', true,
        'feedback_id', feedback_id,
        'correction_date', NOW()
    )
    WHERE id = p_transaction_id;
    
    RETURN feedback_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Grant execute permissions
GRANT EXECUTE ON FUNCTION update_pattern_usage TO authenticated;
GRANT EXECUTE ON FUNCTION get_similar_merchant_patterns TO authenticated;
GRANT EXECUTE ON FUNCTION record_categorization_feedback TO authenticated;

-- Create view for categorization statistics
CREATE VIEW categorization_stats AS
SELECT 
    t.organization_id,
    COUNT(*) as total_transactions,
    COUNT(t.category_id) as categorized_transactions,
    COUNT(CASE WHEN t.categorization_metadata->>'user_corrected' = 'true' THEN 1 END) as user_corrected,
    AVG((t.categorization_metadata->>'confidence_score')::numeric) as avg_confidence,
    COUNT(CASE WHEN t.categorization_metadata->>'categorization_method' = 'rule_based' THEN 1 END) as rule_based_count,
    COUNT(CASE WHEN t.categorization_metadata->>'categorization_method' = 'pattern_matching' THEN 1 END) as pattern_matching_count,
    COUNT(CASE WHEN t.categorization_metadata->>'categorization_method' = 'llm_batch' THEN 1 END) as llm_batch_count
FROM transactions t
WHERE t.created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY t.organization_id;

-- Grant access to view
GRANT SELECT ON categorization_stats TO authenticated;