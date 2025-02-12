use crate::{error::SyncError, settings::Settings};
use chrono::NaiveDate;
use indicatif::{ProgressBar, ProgressStyle};
use llm::{
    builder::{LLMBackend, LLMBuilder},
    chat::{ChatMessage, ChatRole},
};
use std::str::FromStr;

// Helper function to build the LLM client, build the chat message, and get the response.
pub async fn process_llm(
    settings: &Settings,
    billing_period: (NaiveDate, NaiveDate),
    transactions_formatted: &str,
) -> Result<String, SyncError> {
    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .tick_chars("⠁⠂⠄⡀⢀⠠⠐⠈")
            .template("{spinner:.blue} {msg}")
            .expect("Failed to create spinner template"),
    );
    spinner.set_message("Analyzing transactions...");

    let llm = LLMBuilder::new()
        .backend(LLMBackend::from_str(&settings.openai_backend).unwrap())
        .system("You're a helpful assistant that creates a summary of expenses in the last month.")
        .api_key(settings.openai_api_key.clone())
        .model(settings.openai_model.clone())
        .timeout_seconds(1200)
        .temperature(0.7)
        .stream(false)
        .build()
        .map_err(|e| SyncError::LLMError(e.to_string()))?;

    let message = ChatMessage {
        role: ChatRole::User,
        content: format!(
            "
Analyze these financial transactions and create a concise summary (max 100 words) with the following sections:

1. Billing Period: {} to {}
2. Total Expenses: Sum of all purchases (exclude payments, credits, and refunds)
3. Major Categories: Group expenses by category with totals
4. Largest Expenses: List the top 3 individual expenses

Note: Ignore all payments, credits, and refunds in the analysis.

Transactions: 
{}",
            billing_period.0, billing_period.1, transactions_formatted
        ),
    };

    println!("Prompt: {}", message.content);

    spinner.finish_and_clear();
    llm.chat(&[message])
        .await
        .map_err(|e| SyncError::LLMError(e.to_string()))
}
