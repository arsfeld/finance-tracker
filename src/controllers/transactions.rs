use axum::debug_handler;
use axum::routing::get;
use loco_rs::prelude::*;

use crate::{models::transactions, views::transactions::TransactionResponse};

/// Lists all transactions for a given account id.
/// Example request: GET /api/accounts/:account_id/transactions
#[debug_handler]
async fn list_transactions(
    State(ctx): State<AppContext>,
    Path(account_id): Path<String>,
) -> Result<Response> {
    let transactions_list = transactions::Model::find_by_account_id(&ctx.db, &account_id).await?;
    let serialized_transactions: Vec<TransactionResponse> = transactions_list
        .into_iter()
        .map(TransactionResponse::from)
        .collect();
    format::json(serialized_transactions)
}

/// Builds routes for accounts controller.
pub fn routes() -> Routes {
    Routes::new()
        .prefix("/api/accounts/:account_id/transactions")
        .add("/", get(list_transactions))
}
