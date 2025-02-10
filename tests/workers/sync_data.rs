use finance_tracker::app::App;
use loco_rs::prelude::*;
use loco_rs::testing;

use finance_tracker::workers::sync::SyncWorker;
use finance_tracker::workers::sync::SyncWorkerArgs;
use serial_test::serial;

#[tokio::test]
#[serial]
async fn test_run_sync_data_worker() {
    let boot = testing::boot_test::<App>().await.unwrap();

    // Execute the worker ensuring that it operates in 'ForegroundBlocking' mode, which prevents the addition of your worker to the background
    assert!(
        SyncWorker::perform_later(&boot.app_context, SyncWorkerArgs {})
            .await
            .is_ok()
    );
    // Include additional assert validations after the execution of the worker
}
