use crate::settings::NotificationType;
use crate::{error::TrackerError, settings::Settings};
use anyhow::Result;
use console::style;
use indicatif::{ProgressBar, ProgressStyle};
use lettre::message::{header::ContentType, Message};
use lettre::{transport::smtp::AsyncSmtpTransport, AsyncTransport, Tokio1Executor};
use simplefin_bridge::models::Transaction;
use tera::{Context, Tera};

// Helper function to create a consistent spinner style
fn create_spinner(msg: &str) -> ProgressBar {
    let spinner = ProgressBar::new_spinner();
    spinner.set_style(
        ProgressStyle::default_spinner()
            .tick_chars("⠁⠂⠄⡀⢀⠠⠐⠈")
            .template("{spinner:.green} {msg}")
            .expect("Failed to create spinner template"),
    );
    spinner.set_message(msg.to_string());
    spinner
}

// Update the SMS sending function to handle rate limiting and provide better feedback
pub async fn send_twilio_sms(settings: &Settings, text: &str) -> Result<(), TrackerError> {
    let client = reqwest::Client::new();
    let twilio_url = format!(
        "https://api.twilio.com/2010-04-01/Accounts/{}/Messages.json",
        settings.twilio_account_sid.as_ref().unwrap()
    );

    let spinner = create_spinner("Sending SMS");

    for to_phone in settings.twilio_to_phones.as_ref().unwrap().split(',') {
        let to_phone = to_phone.trim();
        spinner.set_message(format!("Sending SMS to {to_phone}"));

        // Add delay between messages to prevent rate limiting
        tokio::time::sleep(tokio::time::Duration::from_millis(500)).await;

        let params = [
            (
                "From",
                settings.twilio_from_phone.as_ref().unwrap().to_string(),
            ),
            ("To", to_phone.to_string()),
            ("Body", text.replace("**", "")),
        ];

        let response = client
            .post(&twilio_url)
            .basic_auth(
                settings.twilio_account_sid.as_ref().unwrap(),
                Some(settings.twilio_auth_token.as_ref().unwrap()),
            )
            .form(&params)
            .send()
            .await
            .map_err(|e| {
                TrackerError::TwilioError(format!("Error sending SMS to {to_phone}: {e}"))
            })?;

        if response.status().is_success() {
            spinner.println(format!(
                "{} SMS sent successfully to {to_phone}",
                style("✓").green(),
            ));
        } else {
            let status = response.status();
            let error_body = response.text().await.unwrap_or_default();
            return Err(TrackerError::TwilioError(format!(
                "Failed to send SMS to {to_phone}. Status: {status}, Body: {error_body}"
            )));
        }
    }

    spinner.finish_and_clear();
    Ok(())
}

pub async fn send_email(
    settings: &Settings,
    text: &str,
    transactions: Vec<Transaction>,
) -> Result<(), TrackerError> {
    let spinner = create_spinner("Preparing email");

    // Convert markdown to HTML
    let mut html_output = String::new();
    pulldown_cmark::html::push_html(&mut html_output, pulldown_cmark::Parser::new(text));

    // Render email template
    let tera = Tera::new("templates/**/*").map_err(|e| {
        TrackerError::EmailError(format!("Failed to initialize template engine: {e}"))
    })?;

    let mut context = Context::new();
    context.insert("text", &html_output);
    context.insert("transactions", &transactions);

    let email_html = tera
        .render("email.mjml", &context)
        .map_err(|e| TrackerError::EmailError(format!("Failed to render template: {e}")))?;

    // Process MJML to HTML
    let root = mrml::parse(&email_html)
        .map_err(|e| TrackerError::EmailError(format!("Failed to parse MJML: {e}")))?;
    let email_html = root
        .element
        .render(&mrml::prelude::render::RenderOptions::default())
        .map_err(|e| TrackerError::EmailError(format!("Failed to render MJML: {e}")))?;

    // Build email message
    let from_email = settings
        .mailer_from
        .as_ref()
        .unwrap()
        .parse()
        .map_err(|e| TrackerError::EmailError(format!("Invalid sender email: {e}")))?;
    let to_email = settings
        .mailer_to
        .as_ref()
        .unwrap()
        .parse()
        .map_err(|e| TrackerError::EmailError(format!("Invalid recipient email: {e}")))?;

    let email = Message::builder()
        .from(from_email)
        .to(to_email)
        .subject("Monthly Expense Trackr")
        .header(ContentType::TEXT_HTML)
        .body(email_html)
        .map_err(|e| TrackerError::EmailError(format!("Failed to build email: {e}")))?;

    // Send email
    spinner.set_message(format!(
        "Sending email to {}",
        settings.mailer_to.as_ref().unwrap()
    ));

    let mailer =
        AsyncSmtpTransport::<Tokio1Executor>::from_url(&settings.mailer_url.clone().unwrap())
            .map_err(|e| TrackerError::EmailError(format!("Failed to create SMTP transport: {e}")))?
            .build();

    mailer
        .send(email)
        .await
        .map_err(|e| TrackerError::EmailError(format!("Failed to send email: {e}")))?;

    spinner.println(format!(
        "{} Email sent successfully to {}",
        style("✓").green(),
        settings.mailer_to.clone().unwrap()
    ));
    spinner.finish_and_clear();

    Ok(())
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
) -> Result<(), TrackerError> {
    let spinner = create_spinner("Sending notification via ntfy.sh");

    // Determine server and topic
    let ntfy_server = if settings.ntfy_server.trim().is_empty() {
        "https://ntfy.sh".to_string()
    } else {
        settings.ntfy_server.clone()
    };

    let ntfy_topic = match notification_type {
        NtfyNotificationType::Info => settings.ntfy_topic.as_ref().unwrap().trim().to_string(),
        NtfyNotificationType::Warning => {
            format!("{}-warning", settings.ntfy_topic.as_ref().unwrap().trim())
        }
    };

    let ntfy_url = format!("{ntfy_server}/{ntfy_topic}");

    // Send notification
    let response = reqwest::Client::new()
        .post(&ntfy_url)
        .body(text.to_owned())
        .send()
        .await
        .map_err(|e| TrackerError::NtfyError(format!("Error sending ntfy.sh notification: {e}")))?;

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
        Err(TrackerError::NtfyError(format!(
            "Failed to send ntfy.sh notification. Status: {status}, Body: {error_body}"
        )))
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
    println!("{} Dispatching notifications", style("🔔").bold());

    for notification_type in notification_types {
        match notification_type {
            NotificationType::Ntfy => {
                println!("{} Dispatching ntfy notification", style("🔔").bold());
                if has_ntfy_settings(settings) {
                    send_ntfy_notification(settings, summary, NtfyNotificationType::Info).await?;
                } else {
                    println!("{} Skipping ntfy notification", style("ℹ️").bold());
                }
            }
            NotificationType::Email => {
                println!("{} Dispatching email notification", style("🔔").bold());
                if has_mailer_settings(settings) {
                    // Note: send_email expects to receive the transactions list.
                    // We clone here if needed.
                    send_email(settings, summary, transactions.clone()).await?;
                } else {
                    println!("{} Skipping email notification", style("ℹ️").bold());
                }
            }
            NotificationType::Sms => {
                println!("{} Dispatching SMS notification", style("🔔").bold());
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
