use sea_orm_migration::prelude::*;

#[derive(DeriveMigrationName)]
pub struct Migration;

#[derive(DeriveIden)]
enum Organizations {
    Table,
    Id,
    Domain,
    SfinUrl,
    Name,
    Url,
    CreatedAt,
    UpdatedAt,
}

#[async_trait::async_trait]
impl MigrationTrait for Migration {
    async fn up(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .create_table(
                Table::create()
                    .table(Organizations::Table)
                    .col(
                        ColumnDef::new(Organizations::Id)
                            .string()
                            .not_null()
                            .primary_key(),
                    )
                    .col(ColumnDef::new(Organizations::Domain).string_len(255).null())
                    .col(
                        ColumnDef::new(Organizations::SfinUrl)
                            .string_len(512)
                            .not_null(),
                    )
                    .col(ColumnDef::new(Organizations::Name).string_len(255).null())
                    .col(ColumnDef::new(Organizations::Url).string_len(512).null())
                    .col(
                        ColumnDef::new(Organizations::CreatedAt)
                            .timestamp_with_time_zone()
                            .not_null()
                            .default(Expr::current_timestamp()),
                    )
                    .col(
                        ColumnDef::new(Organizations::UpdatedAt)
                            .timestamp_with_time_zone()
                            .not_null()
                            .default(Expr::current_timestamp()),
                    )
                    .to_owned(),
            )
            .await?;

        // Add indexes for commonly queried fields
        manager
            .create_index(
                Index::create()
                    .name("idx_organizations_domain")
                    .table(Organizations::Table)
                    .col(Organizations::Domain)
                    .to_owned(),
            )
            .await?;

        manager
            .create_index(
                Index::create()
                    .name("idx_organizations_name")
                    .table(Organizations::Table)
                    .col(Organizations::Name)
                    .to_owned(),
            )
            .await
    }

    async fn down(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        // Drop indexes first
        manager
            .drop_index(
                Index::drop()
                    .name("idx_organizations_domain")
                    .table(Organizations::Table)
                    .to_owned(),
            )
            .await?;

        manager
            .drop_index(
                Index::drop()
                    .name("idx_organizations_name")
                    .table(Organizations::Table)
                    .to_owned(),
            )
            .await?;

        // Then drop the table
        manager
            .drop_table(Table::drop().table(Organizations::Table).to_owned())
            .await
    }
}
