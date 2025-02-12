//! DeepSeek API client implementation for chat and completion functionality.
//!
//! This module provides integration with DeepSeek's models through their API.

#[cfg(feature = "deepseek")]
use crate::{
    chat::{ChatMessage, ChatProvider, ChatRole, Tool},
    completion::{CompletionProvider, CompletionRequest, CompletionResponse},
    embedding::EmbeddingProvider,
    error::LLMError,
    LLMProvider,
};
use async_trait::async_trait;
use reqwest::Client;
use serde::{Deserialize, Serialize};

pub struct DeepSeek {
    pub api_key: String,
    pub model: String,
    pub max_tokens: Option<u32>,
    pub temperature: Option<f32>,
    pub system: Option<String>,
    pub timeout_seconds: Option<u64>,
    pub stream: Option<bool>,
    client: Client,
}

#[derive(Serialize)]
struct DeepSeekChatMessage<'a> {
    role: &'a str,
    content: &'a str,
}

#[derive(Serialize)]
struct DeepSeekChatRequest<'a> {
    model: &'a str,
    messages: Vec<DeepSeekChatMessage<'a>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    temperature: Option<f32>,
    stream: bool,
}

#[derive(Deserialize)]
struct DeepSeekChatResponse {
    choices: Vec<DeepSeekChatChoice>,
}

#[derive(Deserialize)]
struct DeepSeekChatChoice {
    message: DeepSeekChatMsg,
}

#[derive(Deserialize)]
struct DeepSeekChatMsg {
    content: String,
}

impl DeepSeek {
    pub fn new(
        api_key: impl Into<String>,
        model: Option<String>,
        max_tokens: Option<u32>,
        temperature: Option<f32>,
        timeout_seconds: Option<u64>,
        system: Option<String>,
        stream: Option<bool>,
    ) -> Self {
        let mut builder = Client::builder();
        if let Some(sec) = timeout_seconds {
            builder = builder.timeout(std::time::Duration::from_secs(sec));
        }
        Self {
            api_key: api_key.into(),
            model: model.unwrap_or("deepseek-chat".to_string()),
            max_tokens,
            temperature,
            system,
            timeout_seconds,
            stream,
            client: builder.build().expect("Failed to build reqwest Client"),
        }
    }
}

#[async_trait]
impl ChatProvider for DeepSeek {
    /// Sends a chat request to DeepSeek's API.
    ///
    /// # Arguments
    ///
    /// * `messages` - The conversation history as a slice of chat messages
    ///
    /// # Returns
    ///
    /// The provider's response text or an error
    async fn chat(&self, messages: &[ChatMessage]) -> Result<String, LLMError> {
        if self.api_key.is_empty() {
            return Err(LLMError::AuthError("Missing DeepSeek API key".to_string()));
        }

        let mut deepseek_msgs: Vec<DeepSeekChatMessage> = messages
            .iter()
            .map(|m| DeepSeekChatMessage {
                role: match m.role {
                    ChatRole::User => "user",
                    ChatRole::Assistant => "assistant",
                },
                content: &m.content,
            })
            .collect();

        if let Some(system) = &self.system {
            deepseek_msgs.insert(
                0,
                DeepSeekChatMessage {
                    role: "system",
                    content: system,
                },
            );
        }

        let body = DeepSeekChatRequest {
            model: &self.model,
            messages: deepseek_msgs,
            temperature: self.temperature,
            stream: self.stream.unwrap_or(false),
        };

        let mut request = self
            .client
            .post("https://api.deepseek.com/v1/chat/completions")
            .bearer_auth(&self.api_key)
            .json(&body);

        if let Some(timeout) = self.timeout_seconds {
            request = request.timeout(std::time::Duration::from_secs(timeout));
        }

        let resp = request.send().await?.error_for_status()?;

        let json_resp: DeepSeekChatResponse = resp.json().await?;
        let first_choice = json_resp.choices.into_iter().next().ok_or_else(|| {
            LLMError::ProviderError("No choices returned by DeepSeek".to_string())
        })?;

        Ok(first_choice.message.content)
    }

    /// Sends a chat request to DeepSeek's API with tools.
    ///
    /// # Arguments
    ///
    /// * `messages` - The conversation history as a slice of chat messages
    /// * `tools` - Optional slice of tools to use in the chat
    ///
    /// # Returns
    ///
    /// The provider's response text or an error
    async fn chat_with_tools(
        &self,
        _messages: &[ChatMessage],
        _tools: Option<&[Tool]>,
    ) -> Result<String, LLMError> {
        todo!()
    }
}

#[async_trait]
impl CompletionProvider for DeepSeek {
    async fn complete(&self, _req: &CompletionRequest) -> Result<CompletionResponse, LLMError> {
        Ok(CompletionResponse {
            text: "DeepSeek completion not implemented.".into(),
        })
    }
}

#[async_trait]
impl EmbeddingProvider for DeepSeek {
    async fn embed(&self, _text: Vec<String>) -> Result<Vec<Vec<f32>>, LLMError> {
        Err(LLMError::ProviderError(
            "Embedding not supported".to_string(),
        ))
    }
}

impl LLMProvider for DeepSeek {}
