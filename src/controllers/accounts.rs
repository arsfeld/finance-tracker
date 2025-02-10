use axum::debug_handler;
use axum::extract::Query;
use axum::routing::get;
use loco_rs::prelude::*;
use serde::{Deserialize, Serialize};

use crate::{
    models::accounts,
    views::accounts::AccountResponse,
};

/// Query parameters used to filter accounts by organization id.
#[derive(Debug, Deserialize, Serialize)]
pub struct ListQuery {
    pub organization_id: Option<String>,
}

/// Lists all accounts for a given organization id.
/// Example request: GET /api/accounts/?organization_id=org123
#[debug_handler]
async fn list_accounts(
    State(ctx): State<AppContext>,
    Query(query): Query<ListQuery>,
) -> Result<Response> {
    let accounts_list = if query.organization_id.is_none() {
        accounts::Model::find(&ctx.db).await?
    } else {
        accounts::Model::find_by_organization_id(&ctx.db, &query.organization_id.unwrap()).await?
    };
    let serialized_accounts: Vec<AccountResponse> = accounts_list.into_iter().map(AccountResponse::from).collect();
    format::json(serialized_accounts)
}

/// Builds routes for accounts controller.
pub fn routes() -> Routes {
    Routes::new()
        .prefix("/api/accounts")
        .add("/", get(list_accounts))
}
