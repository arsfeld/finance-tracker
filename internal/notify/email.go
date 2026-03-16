package notify

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/rs/zerolog/log"

	"finance_tracker/internal/models"
)

// SendEmail sends an HTML email notification via SMTP.
func SendEmail(mailerURL, from, to, message string, transactions []models.DBTransaction) error {
	if mailerURL == "" || from == "" || to == "" {
		return nil
	}

	htmlContent, err := generateEmailHTML(message, transactions)
	if err != nil {
		return fmt.Errorf("error generating HTML: %w", err)
	}

	mailURL, err := url.Parse(mailerURL)
	if err != nil {
		return fmt.Errorf("error parsing SMTP URL: %w", err)
	}

	smtpHost := mailURL.Hostname()
	smtpPort := mailURL.Port()
	if smtpPort == "" {
		smtpPort = "587"
	}

	username := ""
	password := ""
	if mailURL.User != nil {
		username = mailURL.User.Username()
		if pass, ok := mailURL.User.Password(); ok {
			password = pass
		}
	}

	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      "Finance Tracker - Transaction Summary",
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlContent)

	auth := smtp.PlainAuth("", username, password, smtpHost)
	if err := smtp.SendMail(fmt.Sprintf("%s:%s", smtpHost, smtpPort), auth, from, []string{to}, []byte(msg.String())); err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	log.Info().Str("to", to).Msg("Email notification sent")
	return nil
}

func generateEmailHTML(message string, transactions []models.DBTransaction) (string, error) {
	messageHTML := convertMarkdownToHTML(message)

	const emailTmpl = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.0; color: #2a2a2a; margin: 0; padding: 0; background-color: #f0f7f4; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #fff; padding: 20px; border-radius: 16px; margin-bottom: 20px; text-align: center; }
        .logo { width: 200px; height: 200px; margin-bottom: 20px; }
        .title { color: #2e7d32; font-size: 28px; font-weight: bold; margin-bottom: 20px; }
        .content { background-color: #fff; padding: 20px; border-radius: 16px; margin-bottom: 20px; }
        .message { margin-bottom: 20px; white-space: pre-wrap; }
        .transactions { width: 100%; border-collapse: collapse; margin-top: 20px; }
        .transactions th { background-color: #2e7d32; color: white; padding: 12px; text-align: left; }
        .transactions td { padding: 12px; border-bottom: 1px solid #e8f5e9; }
        .transactions tr:nth-child(even) { background-color: #f8faf8; }
        .footer { background-color: #e8f5e9; padding: 20px; border-radius: 16px; text-align: center; color: #4a4a4a; font-size: 12px; }
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
                <tr><th>Description</th><th>Amount</th><th>Date</th></tr>
                {{range .Transactions}}
                <tr>
                    <td>{{.Description}}</td>
                    <td>{{printf "%.2f" .Amount}}</td>
                    <td>{{formatDate .TransactedAt .Posted}}</td>
                </tr>
                {{end}}
            </table>
        </div>
        <div class="footer">This is an automated message. Please do not reply to this email.</div>
    </div>
</body>
</html>`

	type emailData struct {
		Message      template.HTML
		Transactions []models.DBTransaction
	}

	funcMap := template.FuncMap{
		"formatDate": func(transactedAt *int64, posted int64) string {
			if transactedAt != nil {
				return time.Unix(*transactedAt, 0).Format("2006-01-02 15:04")
			}
			return time.Unix(posted, 0).Format("2006-01-02 15:04")
		},
	}

	tmpl, err := template.New("email").Funcs(funcMap).Parse(emailTmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, emailData{
		Message:      template.HTML(messageHTML),
		Transactions: transactions,
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func convertMarkdownToHTML(md string) string {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	node := p.Parse([]byte(md))
	opts := html.RendererOptions{Flags: html.CommonFlags | html.HrefTargetBlank}
	renderer := html.NewRenderer(opts)
	result := string(markdown.Render(node, renderer))
	return strings.ReplaceAll(result, "\n", "")
}
