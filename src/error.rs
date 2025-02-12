use thiserror::Error;

#[derive(Error, Debug)]
pub enum SyncError {
    #[error("URL error: {0}")]
    UrlError(#[from] url::ParseError),
    
    #[error("SimpleFin error: {0}")]
    SimpleFinError(String),
    
    #[error("LLM error: {0}")]
    LLMError(String),
    
    #[error("Twilio error: {0}")]
    TwilioError(String),
    
    #[error("Validation error: {0}")]
    ValidationError(String),
} 