use sea_orm_migration::{prelude::*, schema::*};

#[derive(DeriveMigrationName)]
pub struct Migration;

#[derive(DeriveIden)]
enum Accounts {
    Table,
    Id,
    Name,
    Currency,
    Balance,
    AvailableBalance,
    BalanceDate,
    Extra,
    OrganizationId,
}

#[async_trait::async_trait]
impl MigrationTrait for Migration {
    async fn up(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .create_table(
                Table::create()
                    .table(Accounts::Table)
                    .col(
                        ColumnDef::new(Accounts::Id)
                            .string()
                            .not_null()
                            .primary_key(),
                    )
                    .col(ColumnDef::new(Accounts::Name).string().not_null())
                    .col(
                        ColumnDef::new(Accounts::Currency)
                            .char_len(3)  // ISO 4217 currency codes are always 3 characters
                            .not_null()
                    )
                    .col(
                        ColumnDef::new(Accounts::Balance)
                            .decimal_len(16, 4)  // Supports large numbers with 4 decimal places
                            .not_null()
                    )
                    .col(
                        ColumnDef::new(Accounts::AvailableBalance)
                            .decimal_len(16, 4)
                    )
                    .col(ColumnDef::new(Accounts::BalanceDate).big_integer().not_null())
                    .col(ColumnDef::new(Accounts::Extra).json())
                    .col(ColumnDef::new(Accounts::OrganizationId).string().not_null())
                    .foreign_key(
                        ForeignKey::create()
                            .name("fk-accounts-organization_id")
                            .from(Accounts::Table, Accounts::OrganizationId)
                            .to(Organizations::Table, Organizations::Id)
                    )
                    .to_owned(),
            )
            .await
    }

    async fn down(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .drop_table(Table::drop().table(Accounts::Table).to_owned())
            .await
    }
}

// Add this enum for the foreign key reference
#[derive(DeriveIden)]
enum Organizations {
    Table,
    Id,
}

