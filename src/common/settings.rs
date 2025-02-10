use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct SimpleFinBridgeSettings {
    pub url: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct OpenAiSettings {
    pub api_key: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Settings {
    pub simplefin_bridge: Option<SimpleFinBridgeSettings>,
    pub openai: Option<OpenAiSettings>,
}

impl Settings {
    pub fn from_json(json: &serde_json::Value) -> Result<Self, serde_json::Error> {
        serde_json::from_value(json.clone())
    }
}
