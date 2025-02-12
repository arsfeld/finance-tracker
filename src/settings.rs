use envconfig::Envconfig;

#[derive(Envconfig)]
pub struct Settings {
    #[envconfig(from = "SIMPLEFIN_BRIDGE_URL")]
    pub simplefin_bridge_url: String,
    #[envconfig(from = "TWILIO_ACCOUNT_SID")]
    pub twilio_account_sid: String,
    #[envconfig(from = "TWILIO_AUTH_TOKEN")]
    pub twilio_auth_token: String,
    #[envconfig(from = "TWILIO_FROM_PHONE")]
    pub twilio_from_phone: String,
    #[envconfig(from = "TWILIO_TO_PHONES")]
    pub twilio_to_phones: String,
    #[envconfig(from = "OPENAI_BACKEND", default = "anthropic")]
    pub openai_backend: String,
    #[envconfig(from = "OPENAI_API_KEY")]
    pub openai_api_key: String,
    #[envconfig(from = "OPENAI_MODEL", default = "claude-3-5-sonnet-latest")]
    pub openai_model: String,
    #[envconfig(from = "MAILER_FROM")]
    pub mailer_from: String,
    #[envconfig(from = "MAILER_TO")]
    pub mailer_to: String,
    #[envconfig(from = "MAILER_HOST")]
    pub mailer_host: String,
    #[envconfig(from = "MAILER_PORT")]
    pub mailer_port: u16,
    #[envconfig(from = "MAILER_USER")]
    pub mailer_user: String,
    #[envconfig(from = "MAILER_PASSWORD")]
    pub mailer_password: String,
    #[envconfig(from = "NTFY_SERVER", default = "https://ntfy.sh")]
    pub ntfy_server: String,
    #[envconfig(from = "NTFY_TOPIC")]
    pub ntfy_topic: String,
}
