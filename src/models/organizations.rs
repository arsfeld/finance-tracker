use chrono::offset::Local;
use loco_rs::prelude::*;
use serde::{Deserialize, Serialize};

pub use super::_entities::organizations::{self, ActiveModel, Entity, Model};

#[derive(Debug, Deserialize, Serialize, Default)]
pub struct CreateParams {
    pub id: String,
    pub domain: Option<String>,
    pub sfin_url: String,
    pub name: Option<String>,
    pub url: Option<String>,
}

#[derive(Debug, Validate, Deserialize)]
pub struct Validator {
    #[validate(length(min = 1, message = "ID must not be empty"))]
    pub id: String,
    #[validate(length(min = 1, message = "SFIN URL must not be empty"))]
    pub sfin_url: String,
}

impl Validatable for super::_entities::organizations::ActiveModel {
    fn validator(&self) -> Box<dyn Validate> {
        Box::new(Validator {
            id: self.id.as_ref().to_owned(),
            sfin_url: self.sfin_url.as_ref().to_owned(),
        })
    }
}

#[async_trait::async_trait]
impl ActiveModelBehavior for super::_entities::organizations::ActiveModel {
    async fn before_save<C>(self, _db: &C, insert: bool) -> Result<Self, DbErr>
    where
        C: ConnectionTrait,
    {
        self.validate()?;
        if insert {
            let mut this = self;
            this.created_at = ActiveValue::Set(Local::now().into());
            this.updated_at = ActiveValue::Set(Local::now().into());
            Ok(this)
        } else {
            Ok(self)
        }
    }
}

impl super::_entities::organizations::Model {
    /// finds an organization by the provided id
    ///
    /// # Errors
    ///
    /// When could not find organization or DB query error
    pub async fn find_by_id(db: &DatabaseConnection, id: &str) -> ModelResult<Self> {
        let org = organizations::Entity::find_by_id(id).one(db).await?;
        org.ok_or_else(|| ModelError::EntityNotFound)
    }

    /// finds an organization by the provided domain
    ///
    /// # Errors
    ///
    /// When could not find organization or DB query error
    pub async fn find_by_domain(db: &DatabaseConnection, domain: &str) -> ModelResult<Self> {
        let org = organizations::Entity::find()
            .filter(
                model::query::condition()
                    .eq(organizations::Column::Domain, domain)
                    .build(),
            )
            .one(db)
            .await?;
        org.ok_or_else(|| ModelError::EntityNotFound)
    }

    /// Asynchronously creates an organization and saves it to the database.
    ///
    /// # Errors
    ///
    /// When could not save the organization into the DB
    pub async fn create(db: &DatabaseConnection, params: &CreateParams) -> ModelResult<Self> {
        let txn = db.begin().await?;

        if organizations::Entity::find_by_id(&params.id)
            .one(&txn)
            .await?
            .is_some()
        {
            return Err(ModelError::EntityAlreadyExists {});
        }

        let org = organizations::ActiveModel {
            id: ActiveValue::set(params.id.to_string()),
            domain: ActiveValue::set(params.domain.clone()),
            sfin_url: ActiveValue::set(params.sfin_url.clone()),
            name: ActiveValue::set(params.name.clone()),
            url: ActiveValue::set(params.url.clone()),
            ..Default::default()
        }
        .insert(&txn)
        .await?;

        txn.commit().await?;

        Ok(org)
    }

    pub async fn from_bridge(
        db: &DatabaseConnection,
        params: &simplefin_bridge::models::Organization,
    ) -> ModelResult<Self> {
        let txn = db.begin().await?;

        let id = params.id.as_ref().unwrap();

        let (created, mut active_org) =
            match organizations::Entity::find_by_id(id).one(&txn).await? {
                Some(model) => (false, model.into_active_model()),
                None => (
                    true,
                    organizations::ActiveModel {
                        id: ActiveValue::set(id.to_string()),
                        created_at: ActiveValue::set(Local::now().into()),
                        updated_at: ActiveValue::set(Local::now().into()),
                        ..Default::default()
                    },
                ),
            };

        println!("active_org: {:?}", active_org);

        active_org.domain = ActiveValue::set(params.domain.clone());
        active_org.sfin_url = ActiveValue::set(params.sfin_url.clone());
        active_org.name = ActiveValue::set(params.name.clone());
        active_org.url = ActiveValue::set(params.url.clone());
        active_org.updated_at = ActiveValue::set(Local::now().into());

        println!("active_org: {:?}", active_org);

        let org = if created {
            active_org.insert(&txn).await?
        } else {
            active_org.update(&txn).await?
        };

        println!("org: {:?}", org);

        txn.commit().await?;

        Ok(org)
    }
}
