use thiserror::Error;

#[derive(Error, Debug)]
pub enum TrackerError {
    #[error("Environment configuration error: {0}")]
    EnvConfigError(String),

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

    #[error("Email error: {0}")]
    EmailError(String),

    #[error("Ntfy error: {0}")]
    NtfyError(String),

    #[error("No transactions found")]
    NoTransactionsFound,

    #[error("Notification error: {0}")]
    NotificationError(String),

    #[error("Cache error: {0}")]
    CacheError(String),
}
