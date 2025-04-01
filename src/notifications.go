package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// NtfyMessage represents a message to be sent to the ntfy.sh service
type NtfyMessage struct {
	Message  string `json:"message"`
	Priority string `json:"priority,omitempty"`
}

// sendNtfyNotification sends a notification to the ntfy.sh service with the specified topic
func sendNtfyNotification(settings *Settings, message string, notificationTopic string) error {
	if settings.NtfyTopic == nil || *settings.NtfyTopic == "" {
		return nil
	}

	topic := *settings.NtfyTopic
	if notificationTopic == "warning" && settings.NtfyTopicWarning != nil {
		topic = *settings.NtfyTopicWarning
	}

	// Strip markdown formatting from the message
	plainMessage := stripMarkdown(message)

	url := fmt.Sprintf("%s/%s", settings.NtfyServer, topic)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(plainMessage)))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Title", "💰 Finance Tracker")

	// Update request body with plain text message
	req.Body = io.NopCloser(bytes.NewBuffer([]byte(plainMessage)))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("notification failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// generateEmailHTML generates a beautiful HTML email with the transaction list
func generateEmailHTML(message string, transactions []Transaction) (string, error) {
	// Convert markdown message to HTML
	messageHTML := convertMarkdownToHTML(message)

	const emailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: Arial, sans-serif;
			line-height: 1.0;
            color: #2a2a2a;
            margin: 0;
            padding: 0;
            background-color: #f0f7f4;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background-color: #ffffff;
            padding: 20px;
            border-radius: 16px;
            margin-bottom: 20px;
            text-align: center;
        }
        .logo {
            width: 200px;
            height: 200px;
            margin-bottom: 20px;
        }
        .title {
            color: #2e7d32;
            font-size: 28px;
            font-weight: bold;
            margin-bottom: 20px;
        }
        .content {
            background-color: #ffffff;
            padding: 20px;
            border-radius: 16px;
            margin-bottom: 20px;
        }
        .message {
            margin-bottom: 20px;
            white-space: pre-wrap;
        }
        .transactions {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        .transactions th {
            background-color: #2e7d32;
            color: white;
            padding: 12px;
            text-align: left;
            border-radius: 8px 8px 0 0;
        }
        .transactions td {
            padding: 12px;
            border-bottom: 1px solid #e8f5e9;
        }
        .transactions tr:nth-child(even) {
            background-color: #f8faf8;
        }
        .transactions tr:last-child td {
            border-bottom: none;
        }
        .footer {
            background-color: #e8f5e9;
            padding: 20px;
            border-radius: 16px;
            text-align: center;
            color: #4a4a4a;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <img src="https://raw.githubusercontent.com/arsfeld/finance-tracker/refs/heads/main/logo.jpg" class="logo" alt="Finance Tracker Logo">
            <div class="title">Transaction Summary</div>
        </div>
        
        <div class="content">
            <div class="message">{{.Message}}</div>
            
            <table class="transactions">
                <tr>
                    <th>Description</th>
                    <th>Amount</th>
                    <th>Date</th>
                </tr>
                {{range .Transactions}}
                <tr>
                    <td>{{.Description}}</td>
                    <td>{{.Amount}}</td>
                    <td>{{formatDate .TransactedAt .Posted}}</td>
                </tr>
                {{end}}
            </table>
        </div>
        
        <div class="footer">
            This is an automated message. Please do not reply to this email.
        </div>
    </div>
</body>
</html>`

	type emailData struct {
		Message      template.HTML
		Transactions []Transaction
	}

	funcMap := template.FuncMap{
		"formatDate": func(transactedAt *int64, posted int64) string {
			if transactedAt != nil {
				return time.Unix(*transactedAt, 0).Format("2006-01-02 15:04")
			}
			return time.Unix(posted, 0).Format("2006-01-02 15:04")
		},
	}

	tmpl, err := template.New("email").Funcs(funcMap).Parse(emailTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, emailData{
		Message:      template.HTML(messageHTML),
		Transactions: transactions,
	}); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	log.Debug().Int("html_length", len(buf.String())).
		Str("html_content", buf.String()).
		Msg("HTML content generated successfully")

	return buf.String(), nil
}

// sendEmailNotification sends an email notification using SMTP
func sendEmailNotification(settings *Settings, message string, transactions []Transaction) error {
	log.Debug().Msg("Starting email notification process")

	if settings.MailerURL == nil || *settings.MailerURL == "" ||
		settings.MailerFrom == nil || *settings.MailerFrom == "" ||
		settings.MailerTo == nil || *settings.MailerTo == "" {
		log.Debug().Msg("Email notification skipped - missing required settings")
		return nil
	}

	log.Debug().
		Str("from", *settings.MailerFrom).
		Str("to", *settings.MailerTo).
		Str("url", *settings.MailerURL).
		Int("transaction_count", len(transactions)).
		Msg("Email notification settings validated")

	// Generate HTML content
	htmlContent, err := generateEmailHTML(message, transactions)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate HTML content")
		return fmt.Errorf("error generating HTML: %w", err)
	}
	log.Debug().Int("html_length", len(htmlContent)).Msg("HTML content generated successfully")

	// Parse SMTP server from URL
	mailURL, err := url.Parse(*settings.MailerURL)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse SMTP URL")
		return fmt.Errorf("error parsing SMTP URL: %w", err)
	}

	smtpHost := mailURL.Hostname()
	smtpPort := mailURL.Port()
	if smtpPort == "" {
		smtpPort = "587" // Default to TLS port
	}

	// Extract username and password from URL if present
	username := ""
	password := ""
	if mailURL.User != nil {
		username = mailURL.User.Username()
		if pass, ok := mailURL.User.Password(); ok {
			password = pass
		}
	}

	log.Debug().
		Str("smtp_host", smtpHost).
		Str("smtp_port", smtpPort).
		Str("username", username).
		Msg("SMTP server details parsed")

	// Prepare email headers
	headers := make(map[string]string)
	headers["From"] = *settings.MailerFrom
	headers["To"] = *settings.MailerTo
	headers["Subject"] = "Finance Tracker - Transaction Summary"
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// Build email message
	var messageBuilder strings.Builder
	for key, value := range headers {
		messageBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	messageBuilder.WriteString("\r\n")
	messageBuilder.WriteString(htmlContent)

	log.Debug().Int("message_size", messageBuilder.Len()).Msg("Email message built")

	// Send email using SMTP
	auth := smtp.PlainAuth("", username, password, smtpHost)
	log.Debug().Msg("Attempting to send email via SMTP")

	err = smtp.SendMail(
		fmt.Sprintf("%s:%s", smtpHost, smtpPort),
		auth,
		*settings.MailerFrom,
		[]string{*settings.MailerTo},
		[]byte(messageBuilder.String()),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send email via SMTP")
		return fmt.Errorf("error sending email: %w", err)
	}

	log.Debug().Msg("Email notification sent successfully")
	return nil
}

// sendNotification sends a notification through the specified notification channels
func sendNotification(settings *Settings, message string, allTransactions []Transaction, notificationTopic string, notificationTypes []string) ([]string, error) {
	var successfulChannels []string

	for _, nt := range notificationTypes {
		switch NotificationType(nt) {
		case NotificationTypeNtfy:
			if err := sendNtfyNotification(settings, message, notificationTopic); err != nil {
				return nil, fmt.Errorf("error sending ntfy notification: %w", err)
			}
			if settings.NtfyTopic != nil && *settings.NtfyTopic != "" {
				successfulChannels = append(successfulChannels, fmt.Sprintf("Ntfy: %s", *settings.NtfyTopic))
			}
		case NotificationTypeEmail:
			if err := sendEmailNotification(settings, message, allTransactions); err != nil {
				return nil, fmt.Errorf("error sending email notification: %w", err)
			}
			if settings.MailerTo != nil && *settings.MailerTo != "" {
				successfulChannels = append(successfulChannels, fmt.Sprintf("Email: %s", *settings.MailerTo))
			}
		}
	}

	return successfulChannels, nil
}
