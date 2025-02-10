use async_trait::async_trait;
use chrono::NaiveDate;
use loco_rs::prelude::*;
use sea_orm::{prelude::Decimal, Order, QueryOrder};
use serde::{Deserialize, Serialize};
use serde_json::Value;

pub use super::_entities::transactions::{self, ActiveModel, Entity, Model};
use super::accounts;

#[derive(Debug, Deserialize, Serialize)]
pub struct CreateParams {
    pub id: String,
    pub account_id: String,
    pub posted: i64,
    pub amount: Decimal,
    pub description: String,
    pub transacted_at: Option<i64>,
    pub pending: Option<bool>,
    pub extra: Option<Value>,
}

#[derive(Debug, Validate, Deserialize)]
pub struct Validator {
    #[validate(length(min = 1, message = "ID must not be empty"))]
    pub id: String,
    #[validate(length(min = 1, message = "Account ID must not be empty"))]
    pub account_id: String,
    #[validate(length(min = 1, message = "Description must not be empty"))]
    pub description: String,
}

impl Validatable for super::_entities::transactions::ActiveModel {
    fn validator(&self) -> Box<dyn Validate> {
        Box::new(Validator {
            id: self.id.as_ref().to_owned(),
            account_id: self.account_id.as_ref().to_owned(),
            description: self.description.as_ref().to_owned(),
        })
    }
}

#[async_trait::async_trait]
impl ActiveModelBehavior for super::_entities::transactions::ActiveModel {
    async fn before_save<C>(self, _db: &C, insert: bool) -> Result<Self, DbErr>
    where
        C: ConnectionTrait,
    {
        self.validate()?;
        if insert {
            let mut this = self;
            this.id = ActiveValue::Set(Uuid::new_v4().to_string());
            Ok(this)
        } else {
            Ok(self)
        }
    }
}

impl super::_entities::transactions::Model {
    /// finds all transactions
    ///
    /// # Errors
    ///
    /// When DB query error occurs
    pub async fn find(db: &DatabaseConnection) -> ModelResult<Vec<Self>> {
        let transactions = transactions::Entity::find().all(db).await?;
        Ok(transactions)
    }

    /// finds a transaction by the provided id
    ///
    /// # Errors
    ///
    /// When could not find transaction or DB query error
    pub async fn find_by_id(db: &DatabaseConnection, id: &str) -> ModelResult<Self> {
        let transaction = transactions::Entity::find_by_id(id).one(db).await?;
        transaction.ok_or_else(|| ModelError::EntityNotFound)
    }

    /// finds transactions by account id
    ///
    /// # Errors
    ///
    /// When DB query error occurs
    pub async fn find_by_account_id(
        db: &DatabaseConnection,
        account_id: &str,
    ) -> ModelResult<Vec<Self>> {
        let transactions = transactions::Entity::find()
            .filter(
                model::query::condition()
                    .eq(transactions::Column::AccountId, account_id)
                    .build(),
            )
            .all(db)
            .await?;
        Ok(transactions)
    }

    /// finds transactions by the provided billing period
    ///
    /// # Errors
    ///
    /// When DB query error occurs
    pub async fn find_by_billing_period(
        db: &DatabaseConnection,
        billing_period: (NaiveDate, NaiveDate),
    ) -> ModelResult<Vec<Self>> {
        let transactions = transactions::Entity::find()
            .filter(
                model::query::condition()
                    .between(
                        transactions::Column::Posted,
                        billing_period
                            .0
                            .and_hms_opt(0, 0, 0)
                            .unwrap()
                            .and_utc()
                            .timestamp(),
                        billing_period
                            .1
                            .and_hms_opt(23, 59, 59)
                            .unwrap()
                            .and_utc()
                            .timestamp(),
                    )
                    .build(),
            )
            .order_by(transactions::Column::Posted, Order::Desc)
            .all(db)
            .await?;
        Ok(transactions)
    }

    /// Asynchronously creates a transaction and saves it to the database.
    ///
    /// # Errors
    ///
    /// When could not save the transaction into the DB or account doesn't exist
    pub async fn create(db: &DatabaseConnection, params: &CreateParams) -> ModelResult<Self> {
        let txn = db.begin().await?;

        // Verify account exists
        accounts::Entity::find_by_id(&params.account_id)
            .one(&txn)
            .await?
            .ok_or_else(|| ModelError::EntityNotFound)?;

        // Check if transaction already exists
        if transactions::Entity::find_by_id(&params.id)
            .one(&txn)
            .await?
            .is_some()
        {
            return Err(ModelError::EntityAlreadyExists {});
        }

        let transaction = transactions::ActiveModel {
            id: ActiveValue::set(params.id.to_string()),
            account_id: ActiveValue::set(params.account_id.to_string()),
            posted: ActiveValue::set(params.posted),
            amount: ActiveValue::set(params.amount),
            description: ActiveValue::set(params.description.to_string()),
            transacted_at: ActiveValue::set(params.transacted_at),
            pending: ActiveValue::set(params.pending),
            extra: ActiveValue::set(params.extra.clone()),
        }
        .insert(&txn)
        .await?;

        txn.commit().await?;

        Ok(transaction)
    }

    pub async fn from_bridge(
        db: &DatabaseConnection,
        params: &simplefin_bridge::models::Transaction,
        account_id: &str,
    ) -> ModelResult<Self> {
        let txn = db.begin().await?;

        let id = params.id.clone();

        let (created, mut active_transaction) = match transactions::Entity::find_by_id(id.clone())
            .one(&txn)
            .await?
        {
            Some(model) => (false, model.into_active_model()),
            None => (
                true,
                transactions::ActiveModel {
                    id: ActiveValue::set(id.clone()),
                    ..Default::default()
                },
            ),
        };

        active_transaction.account_id = ActiveValue::set(account_id.to_string());
        active_transaction.posted = ActiveValue::set(params.posted);
        active_transaction.amount = ActiveValue::set(params.amount.clone());
        active_transaction.description = ActiveValue::set(params.description.clone());
        active_transaction.transacted_at = ActiveValue::set(params.transacted_at);
        active_transaction.pending = ActiveValue::set(params.pending);
        active_transaction.extra = ActiveValue::set(params.extra.clone());

        let transaction = if created {
            active_transaction.insert(&txn).await?
        } else {
            active_transaction.update(&txn).await?
        };

        txn.commit().await?;

        Ok(transaction)
    }
}
