use anyhow::Result;
use chrono::{Datelike, Local, NaiveDate};
use clap::Parser;
use console::style;
use dotenv::dotenv;
use envconfig::Envconfig;

mod error;
mod llm;
mod notifications;
mod settings;
mod transactions;

use llm::process_llm;
use notifications::{send_email, send_ntfy_notification, send_twilio_sms};
use settings::Settings;
use transactions::{format_transactions, get_transactions_for_period, validate_billing_period};

#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Skip sending SMS notifications
    #[arg(long)]
    skip_sms: bool,

    /// Skip sending email notifications
    #[arg(long)]
    skip_email: bool,

    /// Skip sending ntfy notifications
    #[arg(long)]
    skip_ntfy: bool,
}

#[must_use]
pub const fn has_twilio_settings(settings: &Settings) -> bool {
    settings.twilio_to_phones.is_some()
        && settings.twilio_account_sid.is_some()
        && settings.twilio_auth_token.is_some()
        && settings.twilio_from_phone.is_some()
}

#[must_use]
pub const fn has_mailer_settings(settings: &Settings) -> bool {
    settings.mailer_to.is_some() && settings.mailer_url.is_some() && settings.mailer_from.is_some()
}

#[must_use]
pub const fn has_ntfy_settings(settings: &Settings) -> bool {
    settings.ntfy_topic.is_some()
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

            if !args.skip_ntfy && has_ntfy_settings(&settings) {
                send_ntfy_notification(&settings, &text).await?;
            } else {
                println!("{} Skipping ntfy notification", style("ℹ️").bold());
            }

            if !args.skip_email && has_mailer_settings(&settings) {
                send_email(&settings, &text, transactions.clone()).await?;
            } else {
                println!("{} Skipping email notification", style("ℹ️").bold());
            }

            if !args.skip_sms && has_twilio_settings(&settings) {
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
