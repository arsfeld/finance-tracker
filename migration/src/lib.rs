#![allow(elided_lifetimes_in_paths)]
#![allow(clippy::wildcard_imports)]
pub use sea_orm_migration::prelude::*;

mod m20220101_000001_users;

mod m20250209_133336_create_organizations_table;
mod m20250209_133407_create_accounts_table;
mod m20250209_134344_create_transactions_table;
pub struct Migrator;

#[async_trait::async_trait]
impl MigratorTrait for Migrator {
    fn migrations() -> Vec<Box<dyn MigrationTrait>> {
        vec![
            // inject-below (do not remove this comment)
            Box::new(m20250209_134344_create_transactions_table::Migration),
            Box::new(m20250209_133336_create_organizations_table::Migration),
            Box::new(m20250209_133407_create_accounts_table::Migration),
            Box::new(m20220101_000001_users::Migration),
            // inject-above (do not remove this comment)
        ]
    }
}
