use axum::{
    extract::State,
    http::{HeaderMap, StatusCode},
    Json,
};
use uuid::Uuid;

use super::types::{ChatRequest, ChatResponse, Choice, Message};
use super::ServerState;
use crate::chat::{ChatMessage, ChatRole};
use crate::chain::{MultiPromptChain, MultiChainStepBuilder, MultiChainStepMode};

/// Handles chat completion requests to the API server
///
/// # Arguments
/// * `state` - Server state containing LLM registry and auth configuration
/// * `headers` - HTTP request headers for authentication
/// * `req` - Chat request containing messages and model specification
///
/// # Returns
/// * `Ok(Json<ChatResponse>)` - Successful chat completion response
/// * `Err((StatusCode, String))` - Error response with status code and message
///
/// # Authentication
/// If server has auth_key configured, validates Bearer token in Authorization header
///
/// # Request Format
/// Model must be specified as "provider:model_name" (e.g. "openai:gpt-4")
///
/// # Response Format
/// Returns standardized chat completion response with:
/// - Unique ID
/// - Timestamp
/// - Model name
/// - Generated message
pub async fn handle_chat(
    State(state): State<ServerState>,
    headers: HeaderMap,
    Json(req): Json<ChatRequest>,
) -> Result<Json<ChatResponse>, (StatusCode, String)> {
    if let Some(key) = &state.auth_key {
        let auth_header = headers.get("Authorization").ok_or((
            StatusCode::UNAUTHORIZED,
            "Missing authorization".to_string(),
        ))?;

        let auth_str = auth_header.to_str().map_err(|_| {
            (
                StatusCode::UNAUTHORIZED,
                "Invalid authorization header".to_string(),
            )
        })?;

        if !auth_str.starts_with("Bearer ") || &auth_str[7..] != key {
            return Err((StatusCode::UNAUTHORIZED, "Invalid API key".to_string()));
        }
    }

    if !req.steps.is_empty() {
        return handle_chain_request(state, req).await;
    }

    let messages: Vec<ChatMessage> = req
        .messages
        .unwrap_or(vec![])
        .into_iter()
        .map(|msg| ChatMessage {
            role: match msg.role.as_str() {
                "user" => ChatRole::User,
                "assistant" => ChatRole::Assistant,
                _ => ChatRole::User,
            },
            content: msg.content,
        })
        .collect();

    let (provider_id, model_name) = req.model
        .as_ref()
        .ok_or((StatusCode::BAD_REQUEST, "Model is required".to_string()))?
        .split_once(':')
        .ok_or((StatusCode::BAD_REQUEST, "Invalid model format".to_string()))?;

    let provider = state.llms.get(provider_id).ok_or((
        StatusCode::BAD_REQUEST,
        format!("Unknown provider: {}", provider_id),
    ))?;

        let response = provider
            .chat(&messages)
            .await
            .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

        return Ok(Json(ChatResponse {
            id: format!("chatcmpl-{}", Uuid::new_v4()),
            object: "chat.completion".to_string(),
            created: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs(),
            model: model_name.to_string(),
            choices: vec![Choice {
                index: 0,
                message: Message {
                    role: "assistant".to_string(),
                    content: response,
                },
                finish_reason: "stop".to_string(),
            }],
        }));

}

/// Handles multi-step chain requests
async fn handle_chain_request(
    state: ServerState,
    req: ChatRequest,
) -> Result<Json<ChatResponse>, (StatusCode, String)> {
    let mut provider_ids = Vec::new();
    let mut chain = MultiPromptChain::new(&state.llms);

    let transform_response = |resp: String, transform: &str| -> String {
        match transform {
            "extract_think" => resp.lines()
                .skip_while(|line| !line.contains("<think>"))
                .take_while(|line| !line.contains("</think>"))
                .map(|line| line.replace("<think>", "").trim().to_string())
                .filter(|line| !line.is_empty())
                .collect::<Vec<_>>()
                .join("\n"),
            "trim_whitespace" => resp.trim().to_string(),
            _ => resp.to_string()
        }
    };

    if let Some(model) = req.model {
        let (provider_id, _) = model.split_once(':').ok_or((
            StatusCode::BAD_REQUEST,
            "Invalid model format".to_string(),
        ))?;
        
        provider_ids.push(provider_id.to_string());
        let messages = req.messages.unwrap_or(vec![]);
        
        chain = chain.step(
            MultiChainStepBuilder::new(MultiChainStepMode::Chat)
                .provider_id(provider_id.to_string())
                .id("initial")
                .template(format!("{}", messages.last().unwrap().content))
                .max_tokens(req.max_tokens.unwrap_or(1000))
                .temperature(req.temperature.unwrap_or(0.7))
                .response_transform({
                    let transform = req.response_transform.unwrap_or_default();
                    move |resp| transform_response(resp, &transform)
                })
                .build()
                .map_err(|e| (StatusCode::BAD_REQUEST, e.to_string()))?
        );
    }

    chain = chain.chain(req.steps.into_iter().map(|step| {
        provider_ids.push(step.provider_id.clone());
        let transform = step.response_transform.unwrap_or_default();
        MultiChainStepBuilder::new(MultiChainStepMode::Chat)
            .provider_id(step.provider_id)
            .id(step.id)
            .template(step.template)
            .temperature(step.temperature.unwrap_or(0.7))
            .max_tokens(step.max_tokens.unwrap_or(1000))
            .response_transform(move |resp| transform_response(resp, &transform))
            .build()
    }).collect::<Result<Vec<_>, _>>()
    .map_err(|e| (StatusCode::BAD_REQUEST, e.to_string()))?);

    let chain_result = chain.run().await.map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;

    let final_response = chain_result.values().last()
        .ok_or((StatusCode::INTERNAL_SERVER_ERROR, "No response generated".to_string()))?;

    Ok(Json(ChatResponse {
        id: format!("chatcmpl-{}", Uuid::new_v4()),
        object: "chat.completion".to_string(),
        created: std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_secs(),
        model: format!("{}", provider_ids.join(",")),
        choices: vec![Choice {
            index: 0,
            message: Message {
                role: "assistant".to_string(),
                content: final_response.to_string(),
            },
            finish_reason: "stop".to_string(),
        }],
    }))
}
