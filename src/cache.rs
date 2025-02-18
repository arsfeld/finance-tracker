use crate::error::TrackerError;
use rust_decimal::Decimal;
use serde::{Deserialize, Serialize};
use std::fs;
use std::fs::File;
use std::path::PathBuf;
#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct AccountBalance {
    pub balance: Decimal,
    pub balance_date: i64,
}

const APP_NAME: &str = "simplefin-tracker";

fn create_app_cache_dir() -> std::io::Result<PathBuf> {
    let cache_dir = dirs::cache_dir().ok_or(std::io::Error::new(
        std::io::ErrorKind::NotFound,
        "Could not find cache directory",
    ))?;
    let app_cache_dir = cache_dir.join(APP_NAME);
    fs::create_dir_all(&app_cache_dir)?;
    Ok(app_cache_dir)
}

pub fn read_from_cache(account_id: &str) -> Result<AccountBalance, TrackerError> {
    let cache_dir = create_app_cache_dir().map_err(|e| TrackerError::CacheError(e.to_string()))?;
    let file_path = cache_dir.join(format!("{account_id}.json"));
    let file = File::open(file_path).map_err(|e| TrackerError::CacheError(e.to_string()))?;
    let account_balance: AccountBalance =
        serde_json::from_reader(file).map_err(|e| TrackerError::CacheError(e.to_string()))?;
    Ok(account_balance)
}

pub fn write_to_cache(
    account_id: &str,
    account_balance: &AccountBalance,
) -> Result<(), TrackerError> {
    let cache_dir = create_app_cache_dir().map_err(|e| TrackerError::CacheError(e.to_string()))?;
    let file_path = cache_dir.join(format!("{account_id}.json"));
    let file = File::create(file_path).map_err(|e| TrackerError::CacheError(e.to_string()))?;
    serde_json::to_writer(file, &account_balance)
        .map_err(|e| TrackerError::CacheError(e.to_string()))?;
    Ok(())
}
