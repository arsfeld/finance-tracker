use chrono::{Datelike, Local, NaiveDate};
use clap::{Parser, ValueEnum};

use crate::error::TrackerError;
use crate::settings::NotificationType;

#[derive(Clone, ValueEnum, Debug, PartialEq, Eq)]
pub enum DateRangeType {
    CurrentMonth,
    LastMonth,
    Last3Months,
    CurrentYear,
    LastYear,
    Custom,
}

#[derive(Parser)]
#[command(author, version, about, long_about = None)]
pub struct Args {
    // Notification types
    #[arg(short, long, value_delimiter = ',', default_value = "sms,email,ntfy")]
    pub notifications: Vec<NotificationType>,

    #[arg(short, long, default_value_t = false)]
    pub disable_notifications: bool,

    #[arg(long, default_value_t = false)]
    pub disable_cache: bool,

    #[arg(long, default_value_t = false)]
    pub verbose: bool,

    #[arg(long, value_enum, default_value_t = DateRangeType::CurrentMonth)]
    pub date_range: DateRangeType,

    #[arg(long, value_parser = parse_date, requires = "end_date")]
    pub start_date: Option<NaiveDate>,

    #[arg(long, value_parser = parse_date, requires = "start_date")]
    pub end_date: Option<NaiveDate>,
}

// Helper function to parse dates from command line
pub fn parse_date(arg: &str) -> Result<NaiveDate, String> {
    NaiveDate::parse_from_str(arg, "%Y-%m-%d")
        .map_err(|e| format!("Invalid date format (use YYYY-MM-DD): {e}"))
}

// Function to calculate date range based on the DateRangeType
pub fn calculate_date_range(
    date_range_type: DateRangeType,
    start_date: Option<NaiveDate>,
    end_date: Option<NaiveDate>,
) -> Result<(NaiveDate, NaiveDate), TrackerError> {
    match date_range_type {
        DateRangeType::CurrentMonth => {
            let now_local = Local::now();
            let today = now_local.date_naive();
            let start_date = NaiveDate::from_ymd_opt(today.year(), today.month(), 1).unwrap();
            Ok((start_date, today))
        }
        DateRangeType::LastMonth => {
            let now_local = Local::now();
            let today = now_local.date_naive();

            // Get the first day of the current month
            let first_day_current_month =
                NaiveDate::from_ymd_opt(today.year(), today.month(), 1).unwrap();

            // Get the last day of the previous month (day before the first day of current month)
            let end_date = first_day_current_month.pred_opt().unwrap();

            // Get the first day of the previous month
            let start_date = NaiveDate::from_ymd_opt(end_date.year(), end_date.month(), 1).unwrap();

            Ok((start_date, end_date))
        }
        DateRangeType::Last3Months => {
            let now_local = Local::now();
            let today = now_local.date_naive();

            // Get date 3 months ago
            let month = today.month();
            let year = today.year() - if month <= 3 { 1 } else { 0 };
            // Handle wrapping around to previous year
            let month_3_months_ago = (month + 9) % 12;
            // Adjust for 0-based indexing in modulo arithmetic
            let month_3_months_ago = if month_3_months_ago == 0 {
                12
            } else {
                month_3_months_ago
            };

            let start_date = NaiveDate::from_ymd_opt(year, month_3_months_ago, 1).unwrap();
            Ok((start_date, today))
        }
        DateRangeType::CurrentYear => {
            let now_local = Local::now();
            let today = now_local.date_naive();
            let start_date = NaiveDate::from_ymd_opt(today.year(), 1, 1).unwrap();
            Ok((start_date, today))
        }
        DateRangeType::LastYear => {
            let now_local = Local::now();
            let last_year = now_local.year() - 1;
            let start_date = NaiveDate::from_ymd_opt(last_year, 1, 1).unwrap();
            let end_date = NaiveDate::from_ymd_opt(last_year, 12, 31).unwrap();
            Ok((start_date, end_date))
        }
        DateRangeType::Custom => {
            if let (Some(start), Some(end)) = (start_date, end_date) {
                Ok((start, end))
            } else {
                Err(TrackerError::InvalidArgument(
                    "Custom date range requires both start_date and end_date".to_string(),
                ))
            }
        }
    }
}
