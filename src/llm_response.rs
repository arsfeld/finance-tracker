use serde::Deserialize;

/// The top-level structure representing the LLM response.
#[derive(Debug, Deserialize)]
pub struct LLMChatResponse {
    pub id: String,
    pub object: String,
    pub created: u64,
    pub model: String,
    pub provider: String,
    pub choices: Vec<Choice>,
    pub usage: Usage,
}

/// Represents each choice returned in the response.
#[derive(Debug, Deserialize)]
pub struct Choice {
    pub finish_reason: String,
    pub index: u32,
    pub logprobs: Option<serde_json::Value>, // Handles the possibility of null or other types.
    pub message: ChatMessage,
    pub native_finish_reason: String,
}

/// Represents the detailed message contained in a choice.
#[derive(Debug, Deserialize)]
pub struct ChatMessage {
    pub content: String,
    pub refusal: Option<serde_json::Value>, // Using serde_json::Value to be generic (could be null or a string).
    pub role: String,
}

/// Represents the usage statistics.
#[derive(Debug, Deserialize)]
pub struct Usage {
    pub completion_tokens: u64,
    pub prompt_tokens: u64,
    pub total_tokens: u64,
}
