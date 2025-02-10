use serde::Serialize;
use sea_orm::prelude::Decimal;
use crate::models::transactions::Model;
use serde_json::Value;

/// A view that serializes the account
#[derive(Debug, Serialize)]
pub struct TransactionResponse {
    pub id: String,
    pub account_id: String,
    pub posted: i64,
    pub amount: Decimal,
    pub description: String,
    pub transacted_at: Option<i64>,
    pub pending: Option<bool>,
    pub extra: Option<Value>,
    // add any additional fields you need
}

impl From<Model> for TransactionResponse {
    fn from(model: Model) -> Self {
        Self {
            id: model.id,
            account_id: model.account_id,
            posted: model.posted,
            amount: model.amount,
            description: model.description,
            transacted_at: model.transacted_at,
            pending: model.pending,
            extra: model.extra,
        }
    }
} 