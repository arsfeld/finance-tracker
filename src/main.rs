use chrono::{Datelike, Local, NaiveDate};
use clap::Parser;
use console::style;
use dotenv::dotenv;
use anyhow::Result;
use envconfig::Envconfig;

mod error;
mod settings;
mod transactions;
mod notifications;
mod llm;

use error::SyncError;
use settings::Settings;
use transactions::{get_transactions_for_period, format_transactions, validate_billing_period};
use notifications::{send_twilio_sms, send_email};
use llm::process_llm;

#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Skip sending SMS notifications
    #[arg(long)]
    skip_sms: bool,

    /// Skip sending email notifications
    #[arg(long)]
    skip_email: bool,
}

#[tokio::main]
async fn main() -> Result<()> {
    println!("{} Loading configuration...", style("🔧").bold());
    dotenv().ok();
    let settings = Settings::init_from_env()?;
    let args = Args::parse();

    let now_local = Local::now();
    let today = now_local.date_naive();
    let start_date = NaiveDate::from_ymd_opt(today.year(), today.month(), 1).unwrap();
    let billing_period = (start_date, today);
    
    validate_billing_period(billing_period.0, billing_period.1)?;

    println!("{} Fetching transactions...", style("📊").bold());
    let transactions = get_transactions_for_period(&settings, billing_period).await?;

    if transactions.is_empty() {
        println!("{} No transactions found", style("ℹ️").bold());
        return Ok(());
    }

    let transactions_formatted = format_transactions(transactions.clone()).await?;

    println!("{} Analyzing transactions with AI...", style("🤖").bold());
    match process_llm(&settings, billing_period, &transactions_formatted).await {
        Ok(text) => {
            println!("\n{} AI Summary:", style("✨").bold());
            println!("{}", style(text.clone()).cyan());

            if !args.skip_email {
                send_email(&settings, &text, transactions.clone()).await?;
            } else {
                println!("{} Skipping email notification", style("ℹ️").bold());
            }
            
            if !args.skip_sms {
                println!("\n{} Sending SMS notifications...", style("📱").bold());
                if let Err(e) = send_twilio_sms(&settings, &text).await {
                    eprintln!("{} SMS error: {}", style("❌").bold(), e);
                }
            } else {
                println!("{} Skipping SMS notification", style("ℹ️").bold());
            }
        }
        Err(e) => eprintln!("{} Chat error: {}", style("❌").bold(), e),
    }

    Ok(())
}
