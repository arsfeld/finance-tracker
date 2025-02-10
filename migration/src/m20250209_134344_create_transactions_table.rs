use sea_orm_migration::{prelude::*, schema::*};

#[derive(DeriveMigrationName)]
pub struct Migration;

#[derive(DeriveIden)]
enum Transactions {
    Table,
    Id,
    Posted,
    Amount,
    Description,
    TransactedAt,
    Pending,
    Extra,
    AccountId,
}

#[derive(DeriveIden)]
enum Accounts {
    Table,
    Id,
}

#[async_trait::async_trait]
impl MigrationTrait for Migration {
    async fn up(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .create_table(
                Table::create()
                    .table(Transactions::Table)
                    .col(
                        ColumnDef::new(Transactions::Id)
                            .string()
                            .not_null()
                            .primary_key(),
                    )
                    .col(ColumnDef::new(Transactions::AccountId).string().not_null())
                    .col(
                        ColumnDef::new(Transactions::Posted)
                            .big_integer()
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(Transactions::Amount)
                            .decimal_len(16, 4)
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(Transactions::Description)
                            .string()
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(Transactions::TransactedAt)
                            .big_integer()
                            .null(),
                    )
                    .col(ColumnDef::new(Transactions::Pending).boolean().null())
                    .col(ColumnDef::new(Transactions::Extra).json().null())
                    .foreign_key(
                        ForeignKey::create()
                            .name("fk_transaction_account")
                            .from(Transactions::Table, Transactions::AccountId)
                            .to(Accounts::Table, Accounts::Id)
                            .on_delete(ForeignKeyAction::Cascade)
                            .on_update(ForeignKeyAction::Cascade),
                    )
                    .to_owned(),
            )
            .await
    }

    async fn down(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .drop_table(Table::drop().table(Transactions::Table).to_owned())
            .await
    }
}
