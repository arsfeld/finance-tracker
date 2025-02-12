use crate::{error::SyncError, settings::Settings};
use chrono::{DateTime, NaiveDate};
use console::style;
use simplefin_bridge::models::Transaction;
use tabled::{builder::Builder, settings::Style};

pub async fn get_transactions_for_period(
    settings: &Settings,
    billing_period: (NaiveDate, NaiveDate),
) -> Result<Vec<Transaction>, SyncError> {
    let url_parsed =
        url::Url::parse(&settings.simplefin_bridge_url).map_err(SyncError::UrlError)?;

    let bridge = simplefin_bridge::SimpleFinBridge::new(url_parsed);

    let info = bridge
        .info()
        .await
        .map_err(|e| SyncError::SimpleFinError(e.to_string()))?;
    println!(
        "{} Connected to SimpleFin Bridge {}",
        style("✓").green(),
        info.versions.join(", ")
    );

    let params = simplefin_bridge::AccountsParams {
        start_date: Some(
            billing_period
                .0
                .and_hms_opt(0, 0, 0)
                .unwrap()
                .and_utc()
                .timestamp(),
        ),
        end_date: Some(
            billing_period
                .1
                .and_hms_opt(23, 59, 59)
                .unwrap()
                .and_utc()
                .timestamp(),
        ),
        account_ids: None,
        balances_only: None,
        pending: None,
    };

    let accounts = bridge
        .accounts(Some(params))
        .await
        .map_err(|e| SyncError::SimpleFinError(e.to_string()))?;

    let transactions = accounts
        .accounts
        .iter()
        .flat_map(|account| account.transactions.clone().unwrap_or_default())
        .collect();

    Ok(transactions)
}

pub async fn format_transactions(transactions: Vec<Transaction>) -> Result<String, SyncError> {
    let mut builder = Builder::default();
    builder.push_record(["Description", "Amount", "Date"]);

    for transaction in transactions {
        let datetime =
            DateTime::from_timestamp(transaction.transacted_at.unwrap_or(transaction.posted), 0)
                .expect("Transacted at timestamp is invalid");
        builder.push_record([
            transaction.description,
            transaction.amount.to_string(),
            datetime.format("%Y-%m-%d").to_string(),
        ]);
    }

    let mut table = builder.build();
    table.with(Style::modern_rounded().remove_horizontal());

    Ok(table.to_string())
}

// Add validation for the billing period
pub fn validate_billing_period(start: NaiveDate, end: NaiveDate) -> Result<(), SyncError> {
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
