use loco_rs::prelude::*;
use crate::common;
use thiserror::Error;
use chrono::{Local, Duration, NaiveDate, Utc, Datelike};
use url;
use crate::models::organizations::Model as OrganizationModel;
use crate::models::accounts::Model as AccountModel;
use crate::models::transactions::Model as TransactionModel;

#[derive(Debug, Error)]
pub enum SyncDataError {
    #[error("SimpleFin error: {0}")]
    SimpleFinError(#[from] simplefin_bridge::SimpleFinError),
    #[error("URL parsing error: {0}")]
    UrlError(#[from] url::ParseError),
}

pub struct SyncData;

#[async_trait]
impl Task for SyncData {
    fn task(&self) -> TaskInfo {
        TaskInfo {
            name: "sync_data".to_string(),
            detail: "Task generator".to_string(),
        }
    }

    async fn run(&self, ctx: &AppContext, _vars: &task::Vars) -> Result<()> {
        println!("Task SyncData generated");

        let settings = common::settings::Settings::from_json(
            ctx.config.settings.as_ref().unwrap()
        )?;

        let url_parsed = url::Url::parse(&settings.simplefin_bridge.unwrap().url)
            .map_err(|e| loco_rs::Error::wrap(Box::new(SyncDataError::UrlError(e))))?;
        let bridge = simplefin_bridge::SimpleFinBridge::new(url_parsed);

        let info = bridge.info()
            .await
            .map_err(|e| loco_rs::Error::wrap(Box::new(SyncDataError::SimpleFinError(e))))?;
        println!("info: {:?}", info);

        let now = Local::now().date_naive();
        let last_month = now - Duration::days(30);
        let start_date = NaiveDate::from_ymd_opt(last_month.year(), last_month.month(), 15)
            .unwrap()
            .and_hms_opt(0, 0, 0)
            .unwrap()
            .and_utc()
            .timestamp();
        let end_date = now
            .and_hms_opt(23, 59, 59)
            .unwrap()
            .and_utc()
            .timestamp();

        let params = simplefin_bridge::AccountsParams {
            start_date: Some(start_date),
            end_date: Some(end_date),
            account_ids: None,
            balances_only: None,
            pending: None,
        };

        let accounts = bridge.accounts(Some(params))
            .await
            .map_err(|e| loco_rs::Error::wrap(Box::new(SyncDataError::SimpleFinError(e))))?;
        
        // Persist accounts to database
        for account in accounts.accounts {
            println!("Processing account id: {:?}", account.id);

            println!("Creating organization id: {:?}", account.org.id);
            OrganizationModel::from_bridge(&ctx.db, &account.org).await?;

            println!("Creating account id: {:?}", account.id);
            AccountModel::from_bridge(&ctx.db, &account).await?;

            if let Some(transactions) = account.transactions {
                for transaction in transactions {
                    println!("Creating transaction id: {:?}", transaction.id);
                    TransactionModel::from_bridge(&ctx.db, &transaction, &account.id).await?;
                }
            }
        }

        Ok(())
    }
}
