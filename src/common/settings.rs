use serde::{Deserialize, Serialize};
use std::fmt;

#[derive(Debug, Serialize, Deserialize)]
pub struct SimpleFinBridgeSettings {
    pub url: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct OpenAiSettings {
    pub api_key: String,
}

#[derive(Debug, Serialize, Deserialize)]
#[serde(untagged)]
pub enum PhoneNumber {
    String(String),
    Number(i64),
}

impl fmt::Display for PhoneNumber {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        match self {
            PhoneNumber::String(s) => write!(f, "{}", s),
            PhoneNumber::Number(n) => write!(f, "{}", n),
        }
    }
}

#[derive(Debug, Serialize, Deserialize)]
pub struct TwilioSettings {
    pub account_sid: String,
    pub auth_token: String,
    pub from_phone: PhoneNumber,
    pub to_phones: Vec<PhoneNumber>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Settings {
    pub tz: String,
    pub simplefin_bridge: Option<SimpleFinBridgeSettings>,
    pub openai: Option<OpenAiSettings>,
    pub twilio: Option<TwilioSettings>,
}

impl Settings {
    pub fn from_json(json: &serde_json::Value) -> Result<Self, serde_json::Error> {
        serde_json::from_value(json.clone())
    }
}
