use std::fmt;

/// Error types that can occur when interacting with LLM providers.
#[derive(Debug)]
pub enum LLMError {
    /// HTTP request/response errors
    HttpError(String),
    /// Authentication and authorization errors
    AuthError(String),
    /// Invalid request parameters or format
    InvalidRequest(String),
    /// Errors returned by the LLM provider
    ProviderError(String),
    /// JSON serialization/deserialization errors
    JsonError(String),
}

impl fmt::Display for LLMError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            LLMError::HttpError(e) => write!(f, "HTTP Error: {}", e),
            LLMError::AuthError(e) => write!(f, "Auth Error: {}", e),
            LLMError::InvalidRequest(e) => write!(f, "Invalid Request: {}", e),
            LLMError::ProviderError(e) => write!(f, "Provider Error: {}", e),
            LLMError::JsonError(e) => write!(f, "JSON Parse Error: {}", e),
        }
    }
}

impl std::error::Error for LLMError {}

/// Converts reqwest HTTP errors into LlmErrors
impl From<reqwest::Error> for LLMError {
    fn from(err: reqwest::Error) -> Self {
        LLMError::HttpError(err.to_string())
    }
}
