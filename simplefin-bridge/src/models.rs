use serde::{Deserialize, Serialize};
use rust_decimal::Decimal;

#[derive(Debug, Deserialize, Serialize, PartialEq)]
pub struct Info {
    pub versions: Vec<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct AccountSet {
    pub errors: Vec<String>,
    pub accounts: Vec<Account>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Account {
    pub org: Organization,
    pub id: String,
    pub name: String,
    pub currency: String,
    pub balance: Decimal,
    #[serde(rename = "available-balance")]
    pub available_balance: Option<Decimal>,
    #[serde(rename = "balance-date")]
    pub balance_date: i64,
    pub transactions: Option<Vec<Transaction>>,
    pub extra: Option<serde_json::Value>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Organization {
    pub domain: Option<String>,
    #[serde(rename = "sfin-url")]
    pub sfin_url: String,
    pub name: Option<String>,
    pub url: Option<String>,
    pub id: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Transaction {
    pub id: String,
    pub posted: i64,
    pub amount: Decimal,
    pub description: String,
    #[serde(rename = "transacted_at")]
    pub transacted_at: Option<i64>,
    pub pending: Option<bool>,
    pub extra: Option<serde_json::Value>,
}
