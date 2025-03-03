use anyhow::Result;
use chrono::{DateTime, Datelike, Local, NaiveDate, Utc};
use clap::Parser;
use console::style;
use dotenv::dotenv;
use envconfig::Envconfig;
use error::TrackerError;
use rust_decimal::Decimal;
use simplefin_bridge::models::{Account, Transaction};

mod cache;
mod error;
mod llm;
mod llm_response;
mod notifications;
mod settings;
mod transactions;

use llm::{get_llm_prompt, get_llm_response};
use notifications::NtfyNotificationType;
use settings::{NotificationType, Settings};
use transactions::{format_transactions, get_transactions_for_period, validate_billing_period};

// Constants
const TWO_DAYS_IN_SECONDS: i64 = 2 * 24 * 60 * 60;

#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Args {
    // Notification types
    #[arg(short, long, value_delimiter = ',', default_value = "sms,email,ntfy")]
    notifications: Vec<NotificationType>,

    #[arg(short, long, default_value_t = false)]
    disable_notifications: bool,

    #[arg(long, default_value_t = false)]
    disable_cache: bool,

    #[arg(long, default_value_t = false)]
    verbose: bool,
}

#[tokio::main]
async fn main() -> Result<(), TrackerError> {
    println!("{} Loading configuration...", style("🔧").bold());
    dotenv().ok();
    let settings =
        Settings::init_from_env().map_err(|e| TrackerError::EnvConfigError(e.to_string()))?;
    let args = Args::parse();

    // Setup billing period (current month)
    let now_local = Local::now();
    let today = now_local.date_naive();
    let start_date = NaiveDate::from_ymd_opt(today.year(), today.month(), 1).unwrap();
    let billing_period = (start_date, today);
    validate_billing_period(billing_period.0, billing_period.1)?;

    // Fetch and filter accounts
    println!("{} Fetching transactions...", style("📊").bold());
    let accounts: Vec<Account> = get_transactions_for_period(&settings, billing_period)
        .await?
        .iter()
        .filter(|account| account.balance != Decimal::from(0))
        .cloned()
        .collect();

    if args.verbose {
        println!("{} Accounts: {:?}", style("🔍").bold(), accounts);
    }

    let transactions: Vec<Transaction> = accounts
        .iter()
        .flat_map(|account| account.transactions.clone().unwrap_or_default())
        .collect();

    // Handle cache
    let cache = if !args.disable_cache {
        cache::read_cache().unwrap_or_default()
    } else {
        cache::Cache::default()
    };

    if args.verbose {
        println!("{} Cache: {:?}", style("🔍").bold(), cache);
    }

    let mut updated_accounts = cache.accounts.clone().unwrap_or_default();
    let mut has_updated_accounts = false;

    // Process accounts
    println!("{} Accounts:", style("💳").bold());
    for account in &accounts {
        println!("{} {} ({})", style("•").green(), account.name, account.id);

        // Format and display last sync time
        let sync_time = DateTime::from_timestamp(account.balance_date, 0)
            .expect("Balance date timestamp is invalid")
            .format("%Y-%m-%d %H:%M:%S");

        println!("{} Last synced at: {}", style("└").dim(), sync_time);

        // Check if sync is too old
        if account.balance_date < (Utc::now().timestamp() - TWO_DAYS_IN_SECONDS) {
            notifications::send_ntfy_notification(
                &settings,
                &format!("Account {} is not synced", account.name),
                NtfyNotificationType::Warning,
            )
            .await?;
        }

        // Check if account has been updated since last cache
        let is_updated = match updated_accounts.get(&account.id) {
            Some(cached_account) => cached_account.balance_date != account.balance_date,
            None => true, // New account
        };

        if is_updated {
            has_updated_accounts = true;
        }

        // Update the cache with current account data
        updated_accounts.insert(
            account.id.clone(),
            cache::Account {
                balance: account.balance,
                balance_date: account.balance_date,
            },
        );
    }

    // Early return conditions
    if !has_updated_accounts {
        println!("{} No updated accounts", style("🔴").bold());
        return Ok(());
    }

    if transactions.is_empty() {
        println!("{} No transactions found", style("🔴").bold());
        return Err(TrackerError::NoTransactionsFound);
    }

    // Check if we can send a message now
    let last_msg_time = cache.last_successful_message.unwrap_or(0);
    if (Utc::now().timestamp() - last_msg_time) < TWO_DAYS_IN_SECONDS {
        let formatted_time = DateTime::from_timestamp(last_msg_time, 0)
            .unwrap()
            .format("%Y-%m-%d %H:%M:%S");

        println!(
            "{} Last message was sent too recently (at {})",
            style("🔴").bold(),
            formatted_time
        );
        return Err(TrackerError::LastMessageTooSoon);
    }

    // Process transactions with AI
    let transactions_formatted = format_transactions(transactions.clone()).await?;
    let prompt = get_llm_prompt(billing_period, &accounts, &transactions_formatted).await?;

    println!("{} Analyzing transactions with AI...", style("🤖").bold());
    match get_llm_response(&settings, prompt, args.verbose).await {
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
                .await
                .map_err(|e| TrackerError::NotificationError(e.to_string()))?;

                // Update cache with successful message timestamp
                cache::write_cache(&cache::Cache {
                    accounts: Some(updated_accounts),
                    last_successful_message: Some(Utc::now().timestamp()),
                })
                .unwrap();
            } else {
                println!("{} Notifications disabled", style("ℹ️").bold());
            }
        }
        Err(e) => {
            eprintln!("{} Chat error: {}", style("❌").bold(), e);
            return Err(TrackerError::LLMError(e.to_string()));
        }
    }

    Ok(())
}
