use chrono::{Datelike, Local, NaiveDate};
use llm::{
    builder::{LLMBackend, LLMBuilder},
    chat::{ChatMessage, ChatRole},
};
use reqwest;
use tabled::{builder::Builder, settings::Style};
use std::str::FromStr;
use envconfig::Envconfig;
use dotenv::dotenv;
use anyhow::Result;
use console::style;
use indicatif::{ProgressBar, ProgressStyle};
use chrono::DateTime;

mod error;
use error::SyncError;

#[derive(Envconfig)]
struct Settings {
    #[envconfig(from = "SIMPLEFIN_BRIDGE_URL")]
    pub simplefin_bridge_url: String,
    #[envconfig(from = "TWILIO_ACCOUNT_SID")]
    pub twilio_account_sid: String,
    #[envconfig(from = "TWILIO_AUTH_TOKEN")]
    pub twilio_auth_token: String,
    #[envconfig(from = "TWILIO_FROM_PHONE")]
    pub twilio_from_phone: String,
    #[envconfig(from = "TWILIO_TO_PHONES")]
    pub twilio_to_phones: String,
    #[envconfig(from = "OPENAI_BACKEND", default = "anthropic")]
    pub openai_backend: String,
    #[envconfig(from = "OPENAI_API_KEY")]
    pub openai_api_key: String,
    #[envconfig(from = "OPENAI_MODEL", default = "claude-3-5-sonnet-latest")]
    pub openai_model: String,
}

#[tokio::main]
async fn main() -> Result<()> {
    println!("{} Loading configuration...", style("🔧").bold());
    dotenv().ok();
    let settings = Settings::init_from_env()?;

    let now_local = Local::now();
    let today = now_local.date_naive();
    let start_date = NaiveDate::from_ymd_opt(today.year(), today.month(), 1).unwrap();
    let billing_period = (start_date, today);
    
    validate_billing_period(billing_period.0, billing_period.1)?;

    println!("{} Fetching transactions...", style("📊").bold());
    let transactions_formatted = get_transactions_for_period(&settings, billing_period).await?;

    if transactions_formatted.is_empty() {
        println!("{} No transactions found", style("ℹ️").bold());
        return Ok(());
    }

    println!("{} Analyzing transactions with AI...", style("🤖").bold());
    match process_llm(&settings, billing_period, &transactions_formatted).await {
        Ok(text) => {
            println!("\n{} AI Summary:", style("✨").bold());
            println!("{}", style(text.clone()).cyan());
            
            println!("\n{} Sending SMS notifications...", style("📱").bold());
            if let Err(e) = send_twilio_sms(&settings, &text).await {
                eprintln!("{} SMS error: {}", style("❌").bold(), e);
            }
        }
        Err(e) => eprintln!("{} Chat error: {}", style("❌").bold(), e),
    }

    Ok(())
}

async fn get_transactions_for_period(
    settings: &Settings,
    billing_period: (NaiveDate, NaiveDate),
) -> Result<String, SyncError> {
    let url_parsed = url::Url::parse(&settings.simplefin_bridge_url)
        .map_err(SyncError::UrlError)?;
    
    let bridge = simplefin_bridge::SimpleFinBridge::new(url_parsed);

    let info = bridge
        .info()
        .await
        .map_err(|e| SyncError::SimpleFinError(e.to_string()))?;
    println!("{} Connected to SimpleFin Bridge {}", style("✓").green(), info.versions.join(", "));

    let params = simplefin_bridge::AccountsParams {
        start_date: Some(billing_period.0.and_hms_opt(0, 0, 0).unwrap().and_utc().timestamp()),
        end_date: Some(billing_period.1.and_hms_opt(23, 59, 59).unwrap().and_utc().timestamp()),
        account_ids: None,
        balances_only: None,
        pending: None,
    };

    let accounts = bridge
        .accounts(Some(params))
        .await
        .map_err(|e| SyncError::SimpleFinError(e.to_string()))?;

    let mut builder = Builder::default();
    builder.push_record(["Description", "Amount", "Date"]);
    
    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .tick_chars("⠁⠂⠄⡀⢀⠠⠐⠈")
            .template("{spinner:.green} {msg}")
            .expect("Failed to create spinner template")
    );

    for account in accounts.accounts {
        spinner.set_message(format!("Processing account: {}", account.id));
        spinner.tick();

        if let Some(transactions) = account.transactions {
            for transaction in transactions {
                let datetime = DateTime::from_timestamp(transaction.posted, 0)
                    .expect("Posted timestamp is invalid");
                builder.push_record([
                    transaction.description,
                    transaction.amount.to_string(),
                    datetime.format("%Y-%m-%d").to_string(),
                ]);
            }
        }
    }

    spinner.finish_with_message("✓ Transaction processing complete");

    let mut table = builder.build();
    table.with(Style::modern_rounded().remove_horizontal());

    Ok(table.to_string())
}

// Update the SMS sending function to handle rate limiting and provide better feedback
async fn send_twilio_sms(settings: &Settings, text: &str) -> Result<(), SyncError> {
    let client = reqwest::Client::new();
    let twilio_url = format!(
        "https://api.twilio.com/2010-04-01/Accounts/{}/Messages.json",
        settings.twilio_account_sid
    );

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .tick_chars("⠁⠂⠄⡀⢀⠠⠐⠈")
            .template("{spinner:.green} {msg}")
            .expect("Failed to create spinner template")
    );

    for to_phone in settings.twilio_to_phones.split(',') {
        spinner.set_message(format!("Sending SMS to {}", to_phone));
        
        // Add delay between messages to prevent rate limiting
        tokio::time::sleep(tokio::time::Duration::from_millis(500)).await;

        let params = [
            ("From", settings.twilio_from_phone.to_string()),
            ("To", to_phone.trim().to_string()),  // Trim whitespace from phone numbers
            ("Body", text.to_owned()),
        ];

        match client
            .post(&twilio_url)
            .basic_auth(&settings.twilio_account_sid, Some(&settings.twilio_auth_token))
            .form(&params)
            .send()
            .await
        {
            Ok(response) => {
                if response.status().is_success() {
                    spinner.println(format!("{} SMS sent successfully to {}", style("✓").green(), to_phone));
                } else {
                    let status = response.status();
                    let error_body = response.text().await.unwrap_or_default();
                    return Err(SyncError::TwilioError(format!(
                        "Failed to send SMS to {}. Status: {}, Body: {}",
                        to_phone,
                        status,
                        error_body
                    )));
                }
            }
            Err(e) => {
                return Err(SyncError::TwilioError(format!(
                    "Error sending SMS to {}: {}",
                    to_phone, e
                )));
            }
        }
    }
    
    spinner.finish_and_clear();
    Ok(())
}

// Helper function to build the LLM client, build the chat message, and get the response.
async fn process_llm(
    settings: &Settings,
    billing_period: (NaiveDate, NaiveDate),
    transactions_formatted: &str,
) -> Result<String, SyncError> {
    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .tick_chars("⠁⠂⠄⡀⢀⠠⠐⠈")
            .template("{spinner:.blue} {msg}")
            .expect("Failed to create spinner template")
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

// Add validation for the billing period
fn validate_billing_period(start: NaiveDate, end: NaiveDate) -> Result<(), SyncError> {
    if start > end {
        return Err(SyncError::ValidationError(
            "Start date cannot be after end date".to_string(),
        ));
    }
    
    let duration = end.signed_duration_since(start);
    if duration.num_days() > 90 {
        return Err(SyncError::ValidationError(
            "Billing period cannot exceed 90 days".to_string(),
        ));
    }
    
    Ok(())
}
