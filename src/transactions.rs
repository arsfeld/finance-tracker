use crate::{error::TrackerError, settings::Settings};
use chrono::{DateTime, NaiveDate, NaiveDateTime, Utc};
use console::style;
use simplefin_bridge::models::{Account, Transaction};
use tabled::{builder::Builder, settings::Style};

pub async fn get_transactions_for_period(
    settings: &Settings,
    billing_period: (NaiveDate, NaiveDate),
) -> Result<Vec<Account>, TrackerError> {
    let url_parsed =
        url::Url::parse(&settings.simplefin_bridge_url).map_err(TrackerError::UrlError)?;

    let bridge = simplefin_bridge::SimpleFinBridge::new(url_parsed);

    let info = bridge
        .info()
        .await
        .map_err(|e| TrackerError::SimpleFinError(e.to_string()))?;
    println!(
        "{} Connected to SimpleFin Bridge {}",
        style("✓").green(),
        info.versions.join(", ")
    );

    // Helper function to convert NaiveDate to timestamp
    let to_timestamp = |date: NaiveDate, hour: u32, min: u32, sec: u32| -> i64 {
        date.and_hms_opt(hour, min, sec)
            .unwrap()
            .and_utc()
            .timestamp()
    };

    let params = simplefin_bridge::AccountsParams {
        start_date: Some(to_timestamp(billing_period.0, 0, 0, 0)),
        end_date: Some(to_timestamp(billing_period.1, 23, 59, 59)),
        account_ids: None,
        balances_only: None,
        pending: None,
    };

    bridge
        .accounts(Some(params))
        .await
        .map_err(|e| TrackerError::SimpleFinError(e.to_string()))
        .map(|accounts| accounts.accounts)
}

pub async fn format_transactions(transactions: Vec<Transaction>) -> Result<String, TrackerError> {
    let mut builder = Builder::default();
    builder.push_record(["Description", "Amount", "Date"]);

    for txn in transactions {
        let timestamp = txn.transacted_at.unwrap_or(txn.posted);
        let date = DateTime::from_timestamp(timestamp, 0)
            .expect("Invalid timestamp")
            .format("%Y-%m-%d")
            .to_string();

        builder.push_record([txn.description, txn.amount.to_string(), date]);
    }

    Ok(builder
        .build()
        .with(Style::modern_rounded().remove_horizontal())
        .to_string())
}

pub fn validate_billing_period(start: NaiveDate, end: NaiveDate) -> Result<(), TrackerError> {
    if start > end {
        return Err(TrackerError::ValidationError(
            "Start date cannot be after end date".to_string(),
        ));
    }

    if end.signed_duration_since(start).num_days() > 90 {
        return Err(TrackerError::ValidationError(
            "Billing period cannot exceed 90 days".to_string(),
        ));
    }
    Ok(())
}
