use anyhow::Result;
use chrono::{DateTime, Datelike, Local, NaiveDate, Utc};
use clap::Parser;
use console::style;
use dotenv::dotenv;
use envconfig::Envconfig;
use rust_decimal::Decimal;
use simplefin_bridge::models::{Account, Transaction};
mod error;
mod llm;
mod notifications;
mod settings;
mod transactions;

use llm::process_llm;
use notifications::NtfyNotificationType;
use settings::{NotificationType, Settings};
use transactions::{format_transactions, get_transactions_for_period, validate_billing_period};

#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Args {
    // Notification types
    #[arg(short, long, value_delimiter = ',', default_value = "sms,email,ntfy")]
    notifications: Vec<NotificationType>,

    #[arg(short, long, default_value_t = false)]
    disable_notifications: bool,
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
    let accounts: Vec<Account> = get_transactions_for_period(&settings, billing_period)
        .await?
        .iter()
        .filter(|account| account.balance != Decimal::from(0))
        .cloned()
        .collect();

    let transactions: Vec<Transaction> = accounts
        .iter()
        .flat_map(|account| account.transactions.clone().unwrap_or_default())
        .collect();

    // Pretty print accounts and transactions
    println!("{} Accounts:", style("💳").bold());
    for account in &accounts {
        println!("{} {} ({})", style("•").green(), account.name, account.id);

        // Last synced at
        println!(
            "{} Last synced at: {}",
            style("└").dim(),
            DateTime::from_timestamp(account.balance_date, 0)
                .expect("Balance date timestamp is invalid")
                .format("%Y-%m-%d %H:%M:%S")
        );

        // If last synced at is more than 2 days ago, send a warning
        if account.balance_date < (Utc::now().timestamp() - 2 * 24 * 60 * 60) {
            notifications::send_ntfy_notification(
                &settings,
                &format!("Account {} is not synced", account.name),
                NtfyNotificationType::Warning,
            )
            .await?;
        }
    }

    if transactions.is_empty() {
        println!("{} No transactions found", style("ℹ️").bold());
        return Ok(());
    }

    let transactions_formatted = format_transactions(transactions.clone()).await?;

    println!("{} Analyzing transactions with AI...", style("🤖").bold());
    match process_llm(
        &settings,
        billing_period,
        &accounts,
        &transactions_formatted,
    )
    .await
    {
        Ok(text) => {
            println!("\n{} AI Summary:", style("✨").bold());
            println!("{}", style(text.clone()).cyan());

            if !args.disable_notifications {
                notifications::dispatch_notifications(
                    &settings,
                    &text,
                    &transactions,
                    &args.notifications,
                )
                .await?;
            } else {
                println!("{} Notifications disabled", style("ℹ️").bold());
            }
        }
        Err(e) => eprintln!("{} Chat error: {}", style("❌").bold(), e),
    }

    Ok(())
}
