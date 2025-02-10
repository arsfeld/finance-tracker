use loco_rs::prelude::*;
use crate::common;
use thiserror::Error;
use chrono::{Local, Duration, NaiveDate, Utc, Datelike};
use crate::models::transactions::Model as TransactionModel;
use llm::{builder::{LLMBackend, LLMBuilder}, chat::{ChatMessage, ChatRole}};

pub struct Categorize;

#[async_trait]
impl Task for Categorize {
    fn task(&self) -> TaskInfo {
        TaskInfo {
            name: "categorize".to_string(),
            detail: "Task generator".to_string(),
        }
    }

    async fn run(&self, ctx: &AppContext, _vars: &task::Vars) -> Result<()> {
        println!("Task Categorize generated");

        let settings = common::settings::Settings::from_json(
            ctx.config.settings.as_ref().unwrap()
        )?;

        // Initialize and configure the LLM client
        let llm = LLMBuilder::new()
            .backend(LLMBackend::Anthropic)
            .system("You're a helpful assistant that creates a summary of expenses in the last month.")
            .api_key(settings.openai.as_ref().unwrap().api_key.clone()) // Set the API key
            .model("claude-3-5-sonnet-latest") 
            .timeout_seconds(1200)
            .temperature(0.7) // Control response randomness (0.0-1.0)
            .stream(false) // Disable streaming responses
            .build()
            .expect("Failed to build LLM");

        // Billing period is the last month if it's after the 15th, otherwise it's the current month until today
        let billing_period = (if Local::now().day() <= 15 {
            Local::now().date_naive().with_day(15).unwrap().with_month(Local::now().month() - 1).unwrap()
        } else {
            Local::now().date_naive().with_day(15).unwrap().with_month(Local::now().month()).unwrap()
        }, Local::now().date_naive());

        let transactions = TransactionModel::find_by_billing_period(&ctx.db, billing_period).await?;

        if transactions.is_empty() {
            println!("No transactions found for the billing period {} to {}", billing_period.0, billing_period.1);

            let all_transactions = TransactionModel::find(&ctx.db).await?;

            println!("All transactions: {:?}", all_transactions);

            return Ok(());
        }

        // Neatly format the transactions, with the date, description, amount
        let transactions_formatted = transactions.iter().map(|t| format!("{}: {} - {}", t.posted, t.description, t.amount)).collect::<Vec<String>>().join("\n");

        let message = ChatMessage {
            role: ChatRole::User,
            content: format!("
                The billing period is from {} to {}.
            
                Write a few sentences about the following transactions, focus on:
                - Be concise, don't write more than 100 words
                - main categories, with the total amount
                - the biggest expenses
                - the total amount of money spent
                - don't show payments, credits or refunds

                Create separate sections for total spending, major categories and the biggest expenses.
                
                Transactions: {}", billing_period.0, billing_period.1, transactions_formatted),
        };

        println!("Message: {:?}", message);

        // Send chat request and handle the response
        match llm.chat(&[message]).await {
            Ok(text) => println!("Chat response:\n{}", text),
            Err(e) => eprintln!("Chat error: {}", e),
        }

        Ok(())
    }
}
