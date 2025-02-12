use envconfig::Envconfig;

#[derive(Envconfig)]
pub struct Settings {
    #[envconfig(from = "SIMPLEFIN_BRIDGE_URL")]
    pub simplefin_bridge_url: String,
    #[envconfig(from = "TWILIO_ACCOUNT_SID")]
    pub twilio_account_sid: Option<String>,
    #[envconfig(from = "TWILIO_AUTH_TOKEN")]
    pub twilio_auth_token: Option<String>,
    #[envconfig(from = "TWILIO_FROM_PHONE")]
    pub twilio_from_phone: Option<String>,
    #[envconfig(from = "TWILIO_TO_PHONES")]
    pub twilio_to_phones: Option<String>,
    #[envconfig(from = "OPENAI_BACKEND", default = "anthropic")]
    pub openai_backend: String,
    #[envconfig(from = "OPENAI_API_KEY")]
    pub openai_api_key: String,
    #[envconfig(from = "OPENAI_MODEL", default = "claude-3-5-sonnet-latest")]
    pub openai_model: String,
    #[envconfig(from = "MAILER_URL")]
    pub mailer_url: Option<String>,
    #[envconfig(from = "MAILER_FROM")]
    pub mailer_from: Option<String>,
    #[envconfig(from = "MAILER_TO")]
    pub mailer_to: Option<String>,
    #[envconfig(from = "NTFY_SERVER", default = "https://ntfy.sh")]
    pub ntfy_server: String,
    #[envconfig(from = "NTFY_TOPIC")]
    pub ntfy_topic: Option<String>,
}
