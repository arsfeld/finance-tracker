use loco_rs::cli;
use finance_tracker::app::App;
use migration::Migrator;
use dotenv::dotenv;

#[tokio::main]
async fn main() -> loco_rs::Result<()> {
    dotenv().ok();

    cli::main::<App, Migrator>().await
}
