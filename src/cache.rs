use crate::error::TrackerError;
use rust_decimal::Decimal;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::fs::File;
use std::path::PathBuf;

#[derive(Debug, Deserialize, Serialize, Clone)]
pub struct Account {
    pub balance: Decimal,
    pub balance_date: i64,
}

#[derive(Debug, Deserialize, Serialize, Clone, Default)]
pub struct Cache {
    pub accounts: Option<HashMap<String, Account>>,
    pub last_successful_message: Option<i64>,
}

const APP_NAME: &str = "simplefin-tracker";
const CACHE_FILENAME: &str = "cache.json";

fn create_app_cache_dir() -> std::io::Result<PathBuf> {
    let cache_dir = dirs::cache_dir().ok_or(std::io::Error::new(
        std::io::ErrorKind::NotFound,
        "Could not find cache directory",
    ))?;
    let app_cache_dir = cache_dir.join(APP_NAME);
    fs::create_dir_all(&app_cache_dir)?;
    Ok(app_cache_dir)
}

fn get_cache_path() -> Result<PathBuf, TrackerError> {
    let cache_dir = create_app_cache_dir().map_err(|e| TrackerError::CacheError(e.to_string()))?;
    Ok(cache_dir.join(CACHE_FILENAME))
}

pub fn read_cache() -> Result<Cache, TrackerError> {
    let cache_path = get_cache_path()?;

    // If the cache file doesn't exist yet, return an empty cache
    if !cache_path.exists() {
        return Ok(Cache::default());
    }

    let file = File::open(&cache_path).map_err(|e| TrackerError::CacheError(e.to_string()))?;
    serde_json::from_reader(file).map_err(|e| TrackerError::CacheError(e.to_string()))
}

pub fn write_cache(cache: &Cache) -> Result<(), TrackerError> {
    let cache_path = get_cache_path()?;
    let file = File::create(&cache_path).map_err(|e| TrackerError::CacheError(e.to_string()))?;
    serde_json::to_writer(file, &cache).map_err(|e| TrackerError::CacheError(e.to_string()))
}
