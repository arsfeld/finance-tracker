# LLM Analysis Quality Improvements

**Author**: Claude
**Date**: 2025-10-06
**Status**: Research Complete, Implementation Pending

## Executive Summary

This document outlines research findings and recommendations for improving the quality and consistency of AI-generated financial analysis in the Finance Tracker application. Current implementation uses OpenRouter with basic prompting; improvements focus on prompt engineering techniques and structured outputs.

## Current Implementation Analysis

**File**: `src/llm.go`

### Strengths
- Multi-model fallback with random shuffling
- Clear markdown table formatting for transactions/accounts
- Structured prompt with specific sections
- Good error handling and retry logic

### Limitations
1. **No system message** - Missing role/behavior context
2. **No temperature control** - Using API defaults (often 1.0)
3. **No structured output enforcement** - Relies on prompt instructions only
4. **No few-shot examples** - Model learns from scratch each time
5. **Reasoning excluded** - Could benefit from chain-of-thought for complex analysis

## Research Findings

### 1. Prompt Engineering Techniques

#### Temperature Control
- **Recommendation**: Use **0.3-0.5** for financial analysis
- **Rationale**: Lower temperature (0.0-0.5) produces more deterministic, fact-based responses
- **Current**: No explicit setting (likely defaults to 1.0)
- **Impact**: Reduces hallucination and improves consistency

#### System Prompts
- **Recommendation**: Add system message with role-based instruction
- **Example**: "You are an expert financial analyst specializing in personal finance..."
- **Rationale**: Primes model for domain-specific reasoning
- **Impact**: Better contextual understanding and appropriate tone

#### Few-Shot Learning
- **Recommendation**: Include 1-2 examples of ideal analysis
- **Format**: Show input (transactions) → output (analysis) pairs
- **Rationale**: Demonstrates desired structure, tone, and depth
- **Impact**: More consistent output format and quality

#### Chain-of-Thought Reasoning
- **Recommendation**: Enable reasoning for complex patterns
- **Current**: Explicitly excluded (`Reasoning: { Exclude: true }`)
- **Rationale**: Helps with categorization and trend identification
- **Impact**: Better insights, especially for multi-month analysis

### 2. Structured Output Formats

#### JSON Schema Support
- **OpenRouter Support**: ✅ Available for compatible models
- **Compatible Models**: OpenAI, Nitro, some others
- **Implementation**: Use `response_format` parameter with `json_schema`
- **Benefit**: Guaranteed valid JSON structure for parsing

#### Current Model Compatibility
- Need to verify if configured models support structured outputs
- Can use `require_parameters: true` to filter only compatible providers
- Check models at: https://openrouter.ai/docs/features/structured-outputs

#### Tool/Function Calling
- Alternative to JSON schema
- Uses `tools` parameter with function definitions
- Good for extracting specific data points (totals, categories, alerts)

## Recommendations

### Priority 1: Immediate Improvements (High Impact, Low Risk)

1. **Add Temperature Control**
   - Set `temperature: 0.4` in OpenRouterRequest
   - Balances consistency with natural language variety

2. **Add System Message**
   ```go
   Messages: []Message{
       {
           Role: "system",
           Content: "You are an expert financial analyst...",
       },
       {
           Role: "user",
           Content: prompt,
       },
   }
   ```

3. **Enable Reasoning for Complex Analysis**
   - Change `Reasoning.Exclude: false` for multi-month analysis
   - Keep excluded for simple monthly reports (performance)

### Priority 2: Structured Outputs (Medium Impact, Medium Risk)

4. **Implement JSON Schema Response**
   ```go
   ResponseFormat: {
       Type: "json_schema",
       JsonSchema: {
           Name: "financial_analysis",
           Schema: {
               Type: "object",
               Properties: {
                   summary: { type: "string" },
                   total_expenses: { type: "number" },
                   categories: { type: "array", ... },
                   // ...
               },
               Required: ["summary", "total_expenses", "categories"],
           },
       },
   }
   ```
   - Requires model compatibility check
   - Fallback to current approach for incompatible models

### Priority 3: Advanced Techniques (Lower Priority)

5. **Few-Shot Examples**
   - Add 1-2 example analyses to prompt
   - Store as templates for consistency
   - Updates might require careful curation

6. **Dynamic Model Selection**
   - Filter models by structured output support
   - Use `require_parameters: true` in provider preferences

## Implementation Plan

### Phase 1: Basic Improvements (AC #4)
Implement top 3 techniques from Priority 1:
- [ ] Add temperature parameter (0.4)
- [ ] Add system message with financial analyst role
- [ ] Make reasoning configurable based on analysis complexity

### Phase 2: Structured Outputs (Future)
- [ ] Verify model compatibility with JSON schema
- [ ] Implement response_format with schema
- [ ] Add fallback handling for incompatible models
- [ ] Update notification formatting to use JSON structure

### Phase 3: Advanced (Future)
- [ ] Create few-shot example templates
- [ ] Implement dynamic model filtering
- [ ] A/B testing framework for prompt variations

## Testing Strategy (AC #5)

### Test Cases
1. **Single month analysis** - Standard billing cycle
2. **Multi-month analysis** - Two billing cycles
3. **High transaction volume** - 100+ transactions
4. **Diverse categories** - Mixed spending patterns
5. **Edge cases** - Zero transactions, single transaction

### Metrics
- **Consistency**: Same input → similar output structure
- **Accuracy**: Correct totals, categorization
- **Relevance**: Useful insights, not generic statements
- **Format compliance**: Follows markdown structure
- **Readability**: Clear, concise, actionable

### Before/After Comparison
- Save outputs from current implementation
- Apply improvements incrementally
- Compare quality metrics
- Document improvement percentage

## References

- [OpenRouter Structured Outputs](https://openrouter.ai/docs/features/structured-outputs)
- [Prompt Engineering Guide - Temperature](https://www.promptingguide.ai/introduction/settings)
- [Few-Shot Prompting](https://www.promptingguide.ai/techniques/fewshot)
- [Prompt Engineering in 2025](https://www.lakera.ai/blog/prompt-engineering-guide)

## Implementation Status

### ✅ Completed (Phase 1)

**File**: `src/llm.go` (lines 75-103)

1. **Temperature Control** ✅
   - Set to `0.4` for balanced consistency and natural language
   - Reduces hallucination and improves factual accuracy

2. **System Message** ✅
   - Added role-based instruction: "You are an expert financial analyst..."
   - Primes model for domain-specific reasoning
   - Located at lines 82-88

3. **Configurable Reasoning** ✅
   - Enabled for complex analysis (multi-month, 3-months, yearly)
   - Disabled for simple monthly reports (performance optimization)
   - Logic in `main.go` lines 289-292

### Code Changes Summary

```go
// Temperature added
Temperature: 0.4,

// System message prepended to messages array
systemMessage := Message{
    Role: "system",
    Content: `You are an expert financial analyst...`,
}

// Reasoning based on analysis complexity
Reasoning: Reasoning{
    Exclude: !isComplexAnalysis,
}
```

### Testing Verification

**Build Status**: ✅ Compiles successfully
**Type Safety**: ✅ All type signatures updated correctly

### Testing Recommendations

When running with real data, compare:

**Before** (no temperature, no system message, reasoning always excluded):
- May be inconsistent between runs
- Generic analysis without financial expertise
- Simpler insights

**After** (temp=0.4, system message, conditional reasoning):
- More consistent responses
- Domain-specific financial insights
- Better categorization and trend identification
- Enhanced analysis for complex date ranges

### Metrics to Monitor

1. **Consistency**: Run same data multiple times, check similarity
2. **Accuracy**: Verify totals, categories, calculations
3. **Insight Quality**: Check for specific vs generic observations
4. **Format Compliance**: Ensure markdown structure is maintained

### Next Steps

1. ✅ Research complete
2. ✅ Documentation created
3. ✅ Implement Phase 1 improvements
4. ✅ Code verified (compiles)
5. 🔄 Monitor quality improvements in production use
