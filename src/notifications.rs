use crate::settings::NotificationType;
use crate::{error::SyncError, settings::Settings};
use anyhow::Result;
use console::style;
use indicatif::{ProgressBar, ProgressStyle};
use lettre::message::{header::ContentType, Message};
use lettre::{transport::smtp::AsyncSmtpTransport, AsyncTransport, Tokio1Executor};
use simplefin_bridge::models::Transaction;
use tera::{Context, Tera}; // Assuming transactions::Transaction is defined here

// Update the SMS sending function to handle rate limiting and provide better feedback
pub async fn send_twilio_sms(settings: &Settings, text: &str) -> Result<(), SyncError> {
    let client = reqwest::Client::new();
    let twilio_url = format!(
        "https://api.twilio.com/2010-04-01/Accounts/{}/Messages.json",
        settings.twilio_account_sid.as_ref().unwrap()
    );

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .tick_chars("⠁⠂⠄⡀⢀⠠⠐⠈")
            .template("{spinner:.green} {msg}")
            .expect("Failed to create spinner template"),
    );

    for to_phone in settings.twilio_to_phones.as_ref().unwrap().split(',') {
        spinner.set_message(format!("Sending SMS to {to_phone}"));

        // Add delay between messages to prevent rate limiting
        tokio::time::sleep(tokio::time::Duration::from_millis(500)).await;

        let params = [
            (
                "From",
                settings.twilio_from_phone.as_ref().unwrap().to_string(),
            ),
            ("To", to_phone.trim().to_string()), // Trim whitespace from phone numbers
            ("Body", text.to_owned()),
        ];

        match client
            .post(&twilio_url)
            .basic_auth(
                settings.twilio_account_sid.as_ref().unwrap(),
                Some(&settings.twilio_auth_token.as_ref().unwrap()),
            )
            .form(&params)
            .send()
            .await
        {
            Ok(response) => {
                if response.status().is_success() {
                    spinner.println(format!(
                        "{} SMS sent successfully to {}",
                        style("✓").green(),
                        to_phone
                    ));
                } else {
                    let status = response.status();
                    let error_body = response.text().await.unwrap_or_default();
                    return Err(SyncError::TwilioError(format!(
                        "Failed to send SMS to {to_phone}. Status: {status}, Body: {error_body}"
                    )));
                }
            }
            Err(e) => {
                return Err(SyncError::TwilioError(format!(
                    "Error sending SMS to {to_phone}: {e}"
                )));
            }
        }
    }

    spinner.finish_and_clear();
    Ok(())
}

pub async fn send_email(
    settings: &Settings,
    text: &str,
    transactions: Vec<Transaction>,
) -> Result<(), SyncError> {
    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .tick_chars("⠁⠂⠄⡀⢀⠠⠐⠈")
            .template("{spinner:.green} {msg}")
            .expect("Failed to create spinner template"),
    );

    let tera = Tera::new("templates/**/*").unwrap();

    let parser = pulldown_cmark::Parser::new(text);
    let mut html_output = String::new();
    pulldown_cmark::html::push_html(&mut html_output, parser);

    let mut context = Context::new();
    context.insert("text", &html_output);
    context.insert("transactions", &transactions);

    let email_html = tera.render("email.mjml", &context).unwrap();

    let root = mrml::parse(&email_html).unwrap();
    let opts = mrml::prelude::render::RenderOptions::default();
    let email_html = root.render(&opts).unwrap();

    // Build the email message using Lettre.
    let email = Message::builder()
        .from(
            settings
                .mailer_from
                .as_ref()
                .unwrap()
                .parse()
                .map_err(|e| {
                    SyncError::EmailError(format!(
                        "Invalid sender email '{:?}': {}",
                        settings.mailer_from, e
                    ))
                })?,
        )
        .to(settings.mailer_to.as_ref().unwrap().parse().map_err(|e| {
            SyncError::EmailError(format!(
                "Invalid recipient email '{:?}': {}",
                settings.mailer_to, e
            ))
        })?)
        .subject("Monthly Expense Trackr")
        .header(ContentType::TEXT_HTML)
        .body(email_html)
        .map_err(|e| SyncError::EmailError(format!("Failed to build email: {e}")))?;

    let mailer_url = settings.mailer_url.clone().unwrap();

    let mailer = AsyncSmtpTransport::<Tokio1Executor>::from_url(&mailer_url)
        .map_err(|e| SyncError::EmailError(format!("Failed to create SMTP transport: {e}")))?
        .build();

    // Send the email and report the outcome.
    match mailer.send(email).await {
        Ok(_) => {
            spinner.println(format!(
                "{} Email sent successfully to {}",
                style("✓").green(),
                settings.mailer_to.clone().unwrap()
            ));
            Ok(())
        }
        Err(e) => Err(SyncError::EmailError(format!("Failed to send email: {e}"))),
    }
}

#[derive(Debug, Clone, Copy)]
pub enum NtfyNotificationType {
    Info,
    Warning,
}

// New function to send notifications via ntfy.sh
pub async fn send_ntfy_notification(
    settings: &Settings,
    text: &str,
    notification_type: NtfyNotificationType,
) -> Result<(), SyncError> {
    let client = reqwest::Client::new();

    // Use settings.ntfy_server if provided, otherwise default to "https://ntfy.sh"
    let ntfy_server = if settings.ntfy_server.trim().is_empty() {
        "https://ntfy.sh".to_string()
    } else {
        settings.ntfy_server.clone()
    };

    // Ensure that the ntfy_topic is set in your settings.
    let ntfy_topic = match notification_type {
        NtfyNotificationType::Info => settings.ntfy_topic.as_ref().unwrap().trim().to_string(),
        NtfyNotificationType::Warning => {
            format!("{}-warning", settings.ntfy_topic.as_ref().unwrap().trim())
        }
    };
    let ntfy_url = format!("{ntfy_server}/{ntfy_topic}");

    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .tick_chars("⠁⠂⠄⡀⢀⠠⠐⠈")
            .template("{spinner:.blue} {msg}")
            .expect("Failed to create spinner template"),
    );
    spinner.set_message("Sending notification via ntfy.sh".to_string());

    // Send the POST request with the given text as the notification body
    match client.post(&ntfy_url).body(text.to_owned()).send().await {
        Ok(response) => {
            if response.status().is_success() {
                spinner.println(format!(
                    "{} ntfy.sh notification sent successfully",
                    style("✓").green()
                ));
                spinner.finish_and_clear();
                Ok(())
            } else {
                let status = response.status();
                let error_body = response.text().await.unwrap_or_default();
                spinner.finish_and_clear();
                Err(SyncError::NtfyError(format!(
                    "Failed to send ntfy.sh notification. Status: {status}, Body: {error_body}"
                )))
            }
        }
        Err(e) => {
            spinner.finish_and_clear();
            Err(SyncError::NtfyError(format!(
                "Error sending ntfy.sh notification: {e}"
            )))
        }
    }
}

// New helper functions moved from main.rs:
pub const fn has_twilio_settings(settings: &Settings) -> bool {
    settings.twilio_to_phones.is_some()
        && settings.twilio_account_sid.is_some()
        && settings.twilio_auth_token.is_some()
        && settings.twilio_from_phone.is_some()
}

pub const fn has_mailer_settings(settings: &Settings) -> bool {
    settings.mailer_to.is_some() && settings.mailer_url.is_some() && settings.mailer_from.is_some()
}

pub const fn has_ntfy_settings(settings: &Settings) -> bool {
    settings.ntfy_topic.is_some()
}

// New function to dispatch all notifications:
pub async fn dispatch_notifications(
    settings: &Settings,
    summary: &str,
    transactions: &Vec<Transaction>,
    notification_types: &[NotificationType],
) -> Result<()> {
    for notification_type in notification_types {
        match notification_type {
            NotificationType::Ntfy => {
                if has_ntfy_settings(settings) {
                    send_ntfy_notification(settings, summary, NtfyNotificationType::Info).await?;
                } else {
                    println!("{} Skipping ntfy notification", style("ℹ️").bold());
                }
            }
            NotificationType::Email => {
                if has_mailer_settings(settings) {
                    // Note: send_email expects to receive the transactions list.
                    // We clone here if needed.
                    send_email(settings, summary, transactions.clone()).await?;
                } else {
                    println!("{} Skipping email notification", style("ℹ️").bold());
                }
            }
            NotificationType::Sms => {
                if has_twilio_settings(settings) {
                    send_twilio_sms(settings, summary).await?;
                } else {
                    println!("{} Skipping SMS notification", style("ℹ️").bold());
                }
            }
        }
    }
    Ok(())
}
