use loco_rs::prelude::*;
use sea_orm::prelude::Decimal;
use serde::{Deserialize, Serialize};
use serde_json::Value;

pub use super::_entities::accounts::{self, ActiveModel, Entity, Model};
use super::organizations;

#[derive(Debug, Deserialize, Serialize)]
pub struct CreateParams {
    pub id: String,
    pub name: String,
    pub currency: String,
    pub balance: Decimal,
    pub available_balance: Decimal,
    pub balance_date: i64,
    pub extra: Option<Value>,
    pub organization_id: String,
}

#[derive(Debug, Validate, Deserialize)]
pub struct Validator {
    #[validate(length(min = 1, message = "ID must not be empty"))]
    pub id: String,
    #[validate(length(min = 1, message = "Name must not be empty"))]
    pub name: String,
    #[validate(length(equal = 3, message = "Currency must be exactly 3 characters"))]
    pub currency: String,
    pub organization_id: String,
}

impl Validatable for super::_entities::accounts::ActiveModel {
    fn validator(&self) -> Box<dyn Validate> {
        Box::new(Validator {
            id: self.id.as_ref().to_owned(),
            name: self.name.as_ref().to_owned(),
            currency: self.currency.as_ref().to_owned(),
            organization_id: self.organization_id.as_ref().to_owned(),
        })
    }
}

#[async_trait::async_trait]
impl ActiveModelBehavior for super::_entities::accounts::ActiveModel {
    async fn before_save<C>(self, _db: &C, _insert: bool) -> Result<Self, DbErr>
    where
        C: ConnectionTrait,
    {
        self.validate()?;
        Ok(self)
    }
}

impl super::_entities::accounts::Model {
    pub async fn find(db: &DatabaseConnection) -> ModelResult<Vec<Self>> {
        let accounts = accounts::Entity::find().all(db).await?;
        Ok(accounts)
    }

    /// finds an account by the provided id
    ///
    /// # Errors
    ///
    /// When could not find account or DB query error
    pub async fn find_by_id(db: &DatabaseConnection, id: &str) -> ModelResult<Self> {
        let account = accounts::Entity::find_by_id(id).one(db).await?;
        account.ok_or_else(|| ModelError::EntityNotFound)
    }

    /// finds accounts by organization id
    ///
    /// # Errors
    ///
    /// When DB query error occurs
    pub async fn find_by_organization_id(
        db: &DatabaseConnection,
        organization_id: &str,
    ) -> ModelResult<Vec<Self>> {
        let accounts = accounts::Entity::find()
            .filter(
                model::query::condition()
                    .eq(accounts::Column::OrganizationId, organization_id)
                    .build(),
            )
            .all(db)
            .await?;
        Ok(accounts)
    }

    /// Asynchronously creates an account and saves it to the database.
    ///
    /// # Errors
    ///
    /// When could not save the account into the DB or organization doesn't exist
    pub async fn create(db: &DatabaseConnection, params: &CreateParams) -> ModelResult<Self> {
        let txn = db.begin().await?;

        // Verify organization exists
        organizations::Entity::find_by_id(&params.organization_id)
            .one(&txn)
            .await?
            .ok_or_else(|| ModelError::EntityNotFound)?;

        // Check if account already exists
        if accounts::Entity::find_by_id(&params.id)
            .one(&txn)
            .await?
            .is_some()
        {
            return Err(ModelError::EntityAlreadyExists {});
        }

        let account = accounts::ActiveModel {
            id: ActiveValue::set(params.id.to_string()),
            name: ActiveValue::set(params.name.to_string()),
            currency: ActiveValue::set(params.currency.to_string()),
            balance: ActiveValue::set(params.balance),
            available_balance: ActiveValue::set(params.available_balance),
            balance_date: ActiveValue::set(params.balance_date),
            extra: ActiveValue::set(params.extra.clone()),
            organization_id: ActiveValue::set(params.organization_id.to_string()),
        }
        .insert(&txn)
        .await?;

        txn.commit().await?;

        Ok(account)
    }

    pub async fn from_bridge(
        db: &DatabaseConnection,
        params: &simplefin_bridge::models::Account,
    ) -> ModelResult<Self> {
        let txn = db.begin().await?;

        let id = params.id.clone();

        let (created, mut active_account) =
            match accounts::Entity::find_by_id(id.clone()).one(&txn).await? {
                Some(model) => (false, model.into_active_model()),
                None => (
                    true,
                    accounts::ActiveModel {
                        id: ActiveValue::set(id.clone()),
                        ..Default::default()
                    },
                ),
            };

        active_account.name = ActiveValue::set(params.name.clone());
        active_account.currency = ActiveValue::set(params.currency.clone());
        active_account.balance = ActiveValue::set(params.balance);
        active_account.available_balance =
            ActiveValue::set(params.available_balance.unwrap());
        active_account.balance_date = ActiveValue::set(params.balance_date);
        active_account.extra = ActiveValue::set(params.extra.clone());
        active_account.organization_id = ActiveValue::set(params.org.id.as_ref().unwrap().clone());

        let account = if created {
            active_account.insert(&txn).await?
        } else {
            active_account.update(&txn).await?
        };

        txn.commit().await?;

        Ok(account)
    }
}
