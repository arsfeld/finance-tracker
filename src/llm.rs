use crate::{error::TrackerError, settings::Settings};
use chrono::Utc;
use chrono::{DateTime, NaiveDate};
use indicatif::{ProgressBar, ProgressStyle};
use serde_json::json;
use simplefin_bridge::models::Account;
use tokio::time::{sleep, Duration};

use crate::llm_response::LLMChatResponse;

// Helper function to build the LLM client, build the chat message, and get the response.
pub async fn get_llm_prompt(
    billing_period: (NaiveDate, NaiveDate),
    accounts: &Vec<Account>,
    transactions_formatted: &str,
) -> Result<String, TrackerError> {
    let mut accounts_formatted = String::new();
    for account in accounts {
        // Format balance_date to a readable string: "YYYY-MM-DD HH:MM:SS"
        let formatted_date = DateTime::<Utc>::from_timestamp(account.balance_date, 0)
            .unwrap()
            .format("%Y-%m-%d %H:%M:%S")
            .to_string();
        accounts_formatted.push_str(&format!(
            " - {}, balance: {}, last synced: {}\n",
            account.name, account.balance, formatted_date
        ));
    }

    let prompt = format!(
        "
## Financial Transaction Analysis
Billing Period: {} to {}

I need a structured analysis of the provided financial transactions. Please create a concise report (max 150 words total) with the following sections:

### Summary
Provide a human-friendly overview of spending patterns during this period. Be specific about trends and notable observations.

### Analysis Breakdown
1. **Total Expenses**: ${{total}} (Sum of all purchases, excluding payments, credits, and refunds)
2. **Major Categories**: List the top 4-5 spending categories with their totals
   - Category 1: ${{amount}}
   - Category 2: ${{amount}}
   - ...
3. **Largest Expenses**: 
   - ${{expense 1}}: ${{amount}} at ${{merchant}} on ${{date}}
   - ${{expense 2}}: ${{amount}} at ${{merchant}} on ${{date}}
   - ${{expense 3}}: ${{amount}} at ${{merchant}} on ${{date}}
4. **Account Status**:
   - ${{account name}}: Balance ${{amount}}, Last synced ${{date}}
   - ...

Notes:
- Consider only outgoing expenses in your analysis (ignore incoming payments, credits, refunds)
- Format all monetary values consistently (e.g., $1,234.56)
- If a category has no transactions, indicate 'No spending in this category'

Accounts Information: 
{}

Transactions: 
{}",
        billing_period.0, billing_period.1, accounts_formatted, transactions_formatted
    );

    Ok(prompt)
}

pub async fn get_llm_response(
    settings: &Settings,
    prompt: String,
    verbose: bool,
) -> Result<String, TrackerError> {
    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .tick_chars("⠁⠂⠄⡀⢀⠠⠐⠈")
            .template("{spinner:.blue} {msg}")
            .expect("Failed to create spinner template"),
    );

    let client = reqwest::Client::new();
    let url = settings.openai_url.clone();

    let payload = json!({
        "model": settings.openai_model,
        "messages": [
            {
                "role": "user",
                "content": prompt
            }
        ]
    });

    let mut headers = reqwest::header::HeaderMap::new();
    headers.insert(
        reqwest::header::AUTHORIZATION,
        reqwest::header::HeaderValue::from_str(&format!("Bearer {}", settings.openai_api_key))
            .map_err(|e| TrackerError::LLMError(e.to_string()))?,
    );

    // New retry loop parameters:
    let mut attempt = 0;
    let max_retries = 50;
    let mut delay_ms = 500;

    let llm_response: LLMChatResponse = loop {
        attempt += 1;
        spinner.set_message(format!("Analyzing transactions... (attempt {attempt})"));

        let response_result = client
            .post(url.clone())
            .headers(headers.clone())
            .json(&payload)
            .send()
            .await;

        match response_result {
            Ok(response) => {
                // Get the entire response body as text once.
                let resp_text = response.text().await.unwrap_or_default();
                match serde_json::from_str::<LLMChatResponse>(&resp_text) {
                    Ok(parsed_response) => break parsed_response,
                    Err(e) => {
                        spinner.println(format!(
                            "Failed to deserialize LLM response: {e}. Response: {resp_text}. Retry attempt {attempt}/{max_retries}"
                        ));
                    }
                }
            }
            Err(e) => {
                spinner.println(format!(
                    "Request failed: {e}. Retry attempt {attempt}/{max_retries}"
                ));
            }
        }

        if attempt >= max_retries {
            spinner.finish_and_clear();
            return Err(TrackerError::LLMError("Max retries reached".to_string()));
        }

        sleep(Duration::from_millis(delay_ms)).await;
        delay_ms *= 2;
    };

    // Pretty print the LLM response
    if verbose {
        println!("LLM Response: {llm_response:#?}");
    }

    if let Some(choice) = llm_response.choices.first() {
        return Ok(choice.message.content.clone());
    }

    Err(TrackerError::LLMError(
        "No choices found in LLM response".into(),
    ))
}
