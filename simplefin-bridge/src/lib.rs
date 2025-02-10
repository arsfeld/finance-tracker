pub mod models;

use base64::prelude::*;
use models::{AccountSet, Info};
use url::Url;

#[derive(Debug, thiserror::Error)]
pub enum SimpleFinError {
    #[error("Failed to decode token: {0}")]
    TokenDecoding(#[from] base64::DecodeError),
    #[error("Invalid UTF-8 in token: {0}")]
    TokenEncoding(#[from] std::string::FromUtf8Error),
    #[error("Invalid URL: {0}")]
    UrlParsing(#[from] url::ParseError),
    #[error("Request failed: {0}")]
    Request(#[from] reqwest::Error),
}

pub type Result<T> = std::result::Result<T, SimpleFinError>;

#[derive(Debug)]
pub struct SimpleFinBridge {
    client: reqwest::Client,
    url: Url,
}

impl SimpleFinBridge {
    /// Creates a new SimpleFinBridge instance from a URL.
    pub fn new(url: impl Into<Url>) -> Self {
        Self {
            client: reqwest::Client::new(),
            url: url.into(),
        }
    }

    /// Creates a new SimpleFinBridge instance from a base64-encoded token.
    ///
    /// # Errors
    ///
    /// Returns `SimpleFinError` if:
    /// - Token is not valid base64
    /// - Token does not contain valid UTF-8
    /// - Token does not contain a valid URL
    /// - Network request fails
    pub async fn from_token(token: &str) -> Result<Self> {
        let claim_url = String::from_utf8(BASE64_STANDARD.decode(token)?)?;
        let client = reqwest::Client::new();
        let access_url = client.post(claim_url).send().await?.text().await?;

        println!("access_url: {}", access_url);

        let parsed_url = Url::parse(&access_url)?;
        Ok(Self::new(parsed_url))
    }

    /// Fetches API info
    pub async fn info(&self) -> Result<Info> {
        let mut info_url = self.url.clone();
        info_url.set_path(&format!("{}/info", info_url.path()));

        Ok(self.client.get(info_url).send().await?.json().await?)
    }

    /// Fetches account information
    pub async fn accounts(&self, params: Option<AccountsParams>) -> Result<AccountSet> {
        let mut accounts_url = self.url.clone();
        accounts_url.set_path(&format!("{}/accounts", accounts_url.path()));

        if let Some(params) = params {
            let query = params.to_query_string();
            accounts_url.set_query(Some(&query));
        }

        println!("accounts_url: {}", accounts_url);

        Ok(self.client.get(accounts_url).send().await?.json().await?)
    }
}

#[derive(Debug, Default, Clone)]
pub struct AccountsParams {
    pub start_date: Option<i64>,
    pub end_date: Option<i64>,
    pub pending: Option<bool>,
    pub account_ids: Option<Vec<String>>,
    pub balances_only: Option<bool>,
}

impl AccountsParams {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn start_date(mut self, date: i64) -> Self {
        self.start_date = Some(date);
        self
    }

    pub fn end_date(mut self, date: i64) -> Self {
        self.end_date = Some(date);
        self
    }

    pub fn pending(mut self, pending: bool) -> Self {
        self.pending = Some(pending);
        self
    }

    pub fn account_ids(mut self, ids: Vec<String>) -> Self {
        self.account_ids = Some(ids);
        self
    }

    pub fn balances_only(mut self, balances_only: bool) -> Self {
        self.balances_only = Some(balances_only);
        self
    }

    fn to_query_string(&self) -> String {
        let mut params = Vec::new();

        if let Some(start_date) = self.start_date {
            params.push(format!("start-date={}", start_date));
        }
        if let Some(end_date) = self.end_date {
            params.push(format!("end-date={}", end_date));
        }
        if let Some(pending) = self.pending {
            if pending {
                params.push("pending=1".to_string());
            }
        }
        if let Some(account_ids) = &self.account_ids {
            for id in account_ids {
                params.push(format!("account={}", id));
            }
        }
        if let Some(balances_only) = self.balances_only {
            if balances_only {
                params.push("balances-only=1".to_string());
            }
        }

        params.join("&")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::{TimeZone, Utc};

    const TEST_TOKEN: &str =
        "aHR0cHM6Ly9iZXRhLWJyaWRnZS5zaW1wbGVmaW4ub3JnL3NpbXBsZWZpbi9jbGFpbS9ERU1P";

    fn timestamp(year: i32, month: u32, day: u32) -> i64 {
        Utc.with_ymd_and_hms(year, month, day, 0, 0, 0)
            .unwrap()
            .timestamp()
    }

    async fn setup_bridge() -> Result<SimpleFinBridge> {
        SimpleFinBridge::from_token(TEST_TOKEN).await
    }

    #[test]
    fn test_new() {
        let url = Url::parse("https://example.com").unwrap();
        let bridge = SimpleFinBridge::new(url);
        assert_eq!(bridge.url.as_str(), "https://example.com/");
    }

    #[tokio::test]
    async fn test_info() -> Result<()> {
        let bridge = setup_bridge().await?;
        let info = bridge.info().await?;
        assert_eq!(info.versions, vec![String::from("1.0")]);
        Ok(())
    }

    #[tokio::test]
    async fn test_accounts_no_params() -> Result<()> {
        let bridge = setup_bridge().await?;
        let accounts = bridge.accounts(None).await?;
        assert!(accounts.errors.is_empty());
        assert!(!accounts.accounts.is_empty());

        // Verify account structure
        let account = &accounts.accounts[0];
        assert!(!account.id.is_empty());
        assert!(!account.name.is_empty());
        assert!(!account.currency.is_empty());
        assert!(account.balance_date > 0);
        Ok(())
    }

    #[tokio::test]
    async fn test_accounts_with_date_range() -> Result<()> {
        let bridge = setup_bridge().await?;
        let params = AccountsParams::new()
            .start_date(timestamp(2020, 1, 1))
            .end_date(timestamp(2020, 12, 31));

        let accounts = bridge.accounts(Some(params)).await?;
        assert!(accounts.errors.is_empty());
        Ok(())
    }

    #[tokio::test]
    async fn test_accounts_with_specific_accounts() -> Result<()> {
        let bridge = setup_bridge().await?;

        // First get all accounts to get valid IDs
        let all_accounts = bridge.accounts(None).await?;
        let account_id = all_accounts.accounts[0].id.clone();

        let params = AccountsParams::new().account_ids(vec![account_id.clone()]);

        let filtered_accounts = bridge.accounts(Some(params)).await?;
        assert!(filtered_accounts.errors.is_empty());
        assert_eq!(filtered_accounts.accounts.len(), 1);
        assert_eq!(filtered_accounts.accounts[0].id, account_id);
        Ok(())
    }

    #[tokio::test]
    async fn test_accounts_balances_only() -> Result<()> {
        let bridge = setup_bridge().await?;
        let params = AccountsParams::new().balances_only(true);

        let accounts = bridge.accounts(Some(params)).await?;
        assert!(accounts.errors.is_empty());
        for account in accounts.accounts {
            assert!(account.transactions.is_none() || account.transactions.unwrap().is_empty());
        }
        Ok(())
    }

    #[test]
    fn test_accounts_params_query_string() {
        let params = AccountsParams::new()
            .start_date(1577836800) // 2020-01-01
            .end_date(1609459199) // 2020-12-31
            .pending(true)
            .account_ids(vec!["123".to_string(), "456".to_string()])
            .balances_only(true);

        let query = params.to_query_string();
        assert!(query.contains("start-date=1577836800"));
        assert!(query.contains("end-date=1609459199"));
        assert!(query.contains("pending=1"));
        assert!(query.contains("account=123"));
        assert!(query.contains("account=456"));
        assert!(query.contains("balances-only=1"));
    }
}
