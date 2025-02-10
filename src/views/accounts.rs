use serde::Serialize;
use sea_orm::prelude::Decimal;
use crate::models::accounts::Model;
use serde_json::Value;

/// A view that serializes the account
#[derive(Debug, Serialize)]
pub struct AccountResponse {
    pub id: String,
    pub name: String,
    pub currency: String,
    pub balance: Decimal,
    pub available_balance: Decimal,
    pub balance_date: i64,
    pub extra: Option<Value>,
    // add any additional fields you need
}

impl From<Model> for AccountResponse {
    fn from(model: Model) -> Self {
        Self {
            id: model.id,
            name: model.name,
            currency: model.currency,
            balance: model.balance,
            available_balance: model.available_balance,
            balance_date: model.balance_date,
            extra: model.extra,
        }
    }
} 