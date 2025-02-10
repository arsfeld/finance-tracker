use crate::common;
use crate::models::transactions::Model as TransactionModel;
use chrono::{Datelike, Local, NaiveDate};
use llm::{
    builder::{LLMBackend, LLMBuilder},
    chat::{ChatMessage, ChatRole},
};
use loco_rs::prelude::*;
use reqwest;

pub struct Summarize;

#[async_trait]
impl Task for Summarize {
    fn task(&self) -> TaskInfo {
        TaskInfo {
            name: "summarize".to_string(),
            detail: "Task generator".to_string(),
        }
    }

    async fn run(&self, ctx: &AppContext, _vars: &task::Vars) -> Result<()> {
        let settings =
            common::settings::Settings::from_json(ctx.config.settings.as_ref().unwrap())?;

        // Calculate the billing period once
        let now_local = Local::now();
        let today = now_local.date_naive();
        let start_date = NaiveDate::from_ymd_opt(today.year(), today.month(), 1).unwrap();
        let billing_period = (start_date, today);

        // Extract transaction processing using the calculated billing period
        let transactions_formatted = get_transactions_for_period(&ctx.db, billing_period).await?;

        if transactions_formatted.is_empty() {
            return Ok(());
        }

        // Use the same billing period for the LLM message
        match process_llm(&settings, billing_period, &transactions_formatted).await {
            Ok(text) => {
                println!("Chat response:\n{}", text);
                if let Some(twilio_config) = settings.twilio.as_ref() {
                    send_twilio_sms(twilio_config, &text).await;
                } else {
                    eprintln!("Twilio settings not configured.");
                }
            }
            Err(e) => eprintln!("Chat error: {}", e),
        }

        Ok(())
    }
}

// New helper function to handle transaction fetching and formatting
async fn get_transactions_for_period(
    db: &DatabaseConnection,
    billing_period: (NaiveDate, NaiveDate),
) -> Result<String> {
    // Fetch transactions for billing period using the provided billing_period parameter
    let transactions = TransactionModel::find_by_billing_period(db, billing_period).await?;
    if transactions.is_empty() {
        println!(
            "No transactions found for the billing period {} to {}",
            billing_period.0, billing_period.1
        );
        return Ok(String::new());
    }

    // Format transactions for the message
    Ok(transactions
        .iter()
        .map(|t| format!("{}: {} - {}", t.posted, t.description, t.amount))
        .collect::<Vec<String>>()
        .join("\n"))
}

// Helper function for sending SMS through Twilio
async fn send_twilio_sms(twilio_config: &crate::common::settings::TwilioSettings, text: &str) {
    let client = reqwest::Client::new();
    let twilio_url = format!(
        "https://api.twilio.com/2010-04-01/Accounts/{}/Messages.json",
        twilio_config.account_sid
    );

    for to_phone in &twilio_config.to_phones {
        let params = [
            ("From", twilio_config.from_phone.to_string()),
            ("To", to_phone.to_string()),
            ("Body", text.to_owned()),
        ];

        match client
            .post(&twilio_url)
            .basic_auth(&twilio_config.account_sid, Some(&twilio_config.auth_token))
            .form(&params)
            .send()
            .await
        {
            Ok(response) => {
                if response.status().is_success() {
                    println!("SMS sent successfully to {}.", to_phone);
                } else {
                    eprintln!(
                        "Failed to send SMS to {}. Status: {}, Body: {:?}",
                        to_phone,
                        response.status(),
                        response.text().await
                    );
                }
            }
            Err(e) => {
                eprintln!("Error sending SMS to {}: {}", to_phone, e);
            }
        }
    }
}

// Helper function to build the LLM client, build the chat message, and get the response.
async fn process_llm(
    settings: &crate::common::settings::Settings,
    billing_period: (NaiveDate, NaiveDate),
    transactions_formatted: &str,
) -> Result<String> {
    let llm = LLMBuilder::new()
        .backend(LLMBackend::Anthropic)
        .system("You're a helpful assistant that creates a summary of expenses in the last month.")
        .api_key(settings.openai.as_ref().unwrap().api_key.clone())
        .model("claude-3-5-sonnet-latest")
        .timeout_seconds(1200)
        .temperature(0.7)
        .stream(false)
        .build()
        .expect("Failed to build LLM");

    let message = ChatMessage {
            role: ChatRole::User,
            content: format!("
                The billing period is from {} to {}.
            
                Write a few sentences about the following transactions, focus on:
                - Be concise, don't write more than 100 words
                - main categories, with the total amount
                - the biggest expenses
                - the total amount of money spent (don't count payments, credits or refunds)
                - don't show payments, credits or refunds

                Create separate sections for total expenses, major categories and the biggest expenses.
                Show the billing period and summarize spending in the period.
                
                Transactions: {}", billing_period.0, billing_period.1, transactions_formatted),
        };

    println!("Prompt: {:?}", message);

    llm.chat(&[message])
        .await
        .map_err(|e| loco_rs::Error::wrap(e))
}
