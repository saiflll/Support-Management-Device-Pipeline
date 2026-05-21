use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct AgentCapability {
    pub name: String,
    pub description: String,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct AgentConfig {
    pub id: String,
    pub name: String,
    pub provider: String,
    pub model: String,
    pub base_url: Option<String>,
    pub api_key: Option<String>,
    pub capabilities: Vec<String>,
    pub temperature: Option<f32>,
    pub system_prompt: Option<String>,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct ChatMessage {
    pub role: String,
    pub content: String,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct AgentRequest {
    pub agent_id: String,
    pub messages: Vec<ChatMessage>,
    pub stream: bool,
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct AgentResponse {
    pub agent_id: String,
    pub content: String,
    pub provider: String,
    pub model: String,
}

#[derive(Clone, Serialize, Deserialize, Debug, Default)]
pub struct AgentUsage {
    pub agent_id: String,
    pub agent_name: String,
    pub provider: String,
    pub model: String,
    pub total_requests: u64,
    pub total_input_tokens: u64,
    pub total_output_tokens: u64,
    pub total_cost: f64,
}

fn model_price(model: &str) -> (f64, f64) {
    let m = model.to_lowercase();
    if m.contains("gpt-4o-mini") {
        (0.00015, 0.0006)
    } else if m.contains("gpt-4o") {
        (0.0025, 0.01)
    } else if m.contains("gpt-4") && !m.contains("turbo") {
        (0.03, 0.06)
    } else if m.contains("gpt-4-turbo") || m.contains("gpt-4-1106") {
        (0.01, 0.03)
    } else if m.contains("gpt-3.5-turbo") {
        (0.0005, 0.0015)
    } else if m.contains("claude-3-opus") || m.contains("claude-opus") {
        (0.015, 0.075)
    } else if m.contains("claude-3-sonnet") || m.contains("claude-sonnet") {
        (0.003, 0.015)
    } else if m.contains("claude-3-haiku") || m.contains("claude-haiku") {
        (0.00025, 0.00125)
    } else if m.contains("gemini-1.5-pro") || m.contains("gemini-pro") {
        (0.00125, 0.005)
    } else if m.contains("gemini-1.5-flash") || m.contains("gemini-flash") {
        (0.000075, 0.0003)
    } else if m.contains("gemini-2.0") || m.contains("gemini-2") {
        (0.0001, 0.0004)
    } else if m.contains("llama") || m.contains("mistral") || m.contains("mixtral") || m.contains("qwen") || m.contains("deepseek") {
        (0.0005, 0.0015)
    } else {
        (0.001, 0.002)
    }
}

pub struct AiManager {
    pub agents: Arc<Mutex<HashMap<String, AgentConfig>>>,
    pub usage: Arc<Mutex<HashMap<String, AgentUsage>>>,
}

impl AiManager {
    pub fn new() -> Self {
        let mut agents = HashMap::new();

        agents.insert(
            "coder".to_string(),
            AgentConfig {
                id: "coder".to_string(),
                name: "Coder".to_string(),
                provider: "openai".to_string(),
                model: "gpt-4o".to_string(),
                base_url: None,
                api_key: None,
                capabilities: vec![
                    "code".to_string(),
                    "debug".to_string(),
                    "refactor".to_string(),
                ],
                temperature: Some(0.2),
                system_prompt: Some(
                    "You are an expert software engineer. Write clean, efficient code.".to_string(),
                ),
            },
        );

        agents.insert(
            "shell".to_string(),
            AgentConfig {
                id: "shell".to_string(),
                name: "Shell".to_string(),
                provider: "openai".to_string(),
                model: "gpt-4o".to_string(),
                base_url: None,
                api_key: None,
                capabilities: vec![
                    "shell".to_string(),
                    "terminal".to_string(),
                    "automation".to_string(),
                ],
                temperature: Some(0.1),
                system_prompt: Some(
                    "You are a shell expert. Provide precise terminal commands.".to_string(),
                ),
            },
        );

        agents.insert(
            "architect".to_string(),
            AgentConfig {
                id: "architect".to_string(),
                name: "Architect".to_string(),
                provider: "anthropic".to_string(),
                model: "claude-sonnet-4-20250514".to_string(),
                base_url: None,
                api_key: None,
                capabilities: vec![
                    "design".to_string(),
                    "planning".to_string(),
                    "architecture".to_string(),
                ],
                temperature: Some(0.4),
                system_prompt: Some(
                    "You are a software architect. Design robust, scalable systems.".to_string(),
                ),
            },
        );

        Self {
            agents: Arc::new(Mutex::new(agents)),
            usage: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    pub fn record_usage(&self, agent_id: &str, input_tokens: u64, output_tokens: u64, cost: f64) {
        let mut usage = self.usage.lock().unwrap();
        let entry = usage.entry(agent_id.to_string()).or_insert_with(|| {
            let agents = self.agents.lock().unwrap();
            let agent = agents.get(agent_id);
            AgentUsage {
                agent_id: agent_id.to_string(),
                agent_name: agent.map(|a| a.name.clone()).unwrap_or_default(),
                provider: agent.map(|a| a.provider.clone()).unwrap_or_default(),
                model: agent.map(|a| a.model.clone()).unwrap_or_default(),
                total_requests: 0,
                total_input_tokens: 0,
                total_output_tokens: 0,
                total_cost: 0.0,
            }
        });
        entry.total_requests += 1;
        entry.total_input_tokens += input_tokens;
        entry.total_output_tokens += output_tokens;
        entry.total_cost += cost;
    }

    pub fn get_usage(&self) -> Vec<AgentUsage> {
        let usage = self.usage.lock().unwrap();
        let mut list: Vec<AgentUsage> = usage.values().cloned().collect();
        list.sort_by(|a, b| b.total_cost.partial_cmp(&a.total_cost).unwrap_or(std::cmp::Ordering::Equal));
        list
    }

    pub fn reset_usage(&self) {
        let mut usage = self.usage.lock().unwrap();
        usage.clear();
    }

    pub fn auto_route(&self, task_type: &str) -> Vec<AgentConfig> {
        let agents = self.agents.lock().unwrap();
        let task_lower = task_type.to_lowercase();

        let mut scored: Vec<(i32, AgentConfig)> = agents
            .values()
            .map(|agent| {
                let score = agent
                    .capabilities
                    .iter()
                    .filter(|cap| task_lower.contains(&cap.to_lowercase()))
                    .count() as i32;
                (score, agent.clone())
            })
            .collect();

        scored.sort_by(|a, b| b.0.cmp(&a.0));
        scored.into_iter().map(|(_, agent)| agent).collect()
    }

    pub fn get_agent(&self, agent_id: &str) -> Option<AgentConfig> {
        let agents = self.agents.lock().unwrap();
        agents.get(agent_id).cloned()
    }

    pub fn list_agents(&self) -> Vec<AgentConfig> {
        let agents = self.agents.lock().unwrap();
        agents.values().cloned().collect()
    }

    pub fn update_agent(&self, config: AgentConfig) -> Result<(), String> {
        let mut agents = self.agents.lock().unwrap();
        agents.insert(config.id.clone(), config);
        Ok(())
    }

    pub fn remove_agent(&self, agent_id: &str) -> Result<(), String> {
        let mut agents = self.agents.lock().unwrap();
        agents.remove(agent_id);
        Ok(())
    }

    pub fn set_provider(
        &self,
        provider: &str,
        base_url: Option<String>,
        api_key: Option<String>,
    ) -> Result<(), String> {
        let mut agents = self.agents.lock().unwrap();
        for (_, agent) in agents.iter_mut() {
            agent.provider = provider.to_string();
            if let Some(ref url) = base_url {
                agent.base_url = Some(url.clone());
            }
            if let Some(ref key) = api_key {
                agent.api_key = Some(key.clone());
            }
        }
        Ok(())
    }
}

fn build_request(agent: &AgentConfig, messages: &[ChatMessage]) -> Result<reqwest::blocking::Request, String> {
    let provider = agent.provider.as_str();
    let known_providers = ["openai", "anthropic", "google", "ollama"];
    if !known_providers.contains(&provider) {
        return Err(format!("Unknown provider: {}", provider));
    }

    let base_url = agent
        .base_url
        .clone()
        .unwrap_or_else(|| match provider {
            "openai" => "https://api.openai.com/v1".to_string(),
            "anthropic" => "https://api.anthropic.com/v1".to_string(),
            "google" => "https://generativelanguage.googleapis.com/v1beta".to_string(),
            "ollama" => "http://localhost:11434/v1".to_string(),
            _ => unreachable!(),
        });

    let client = reqwest::blocking::Client::new();

    match agent.provider.as_str() {
        "anthropic" => {
            let mut sys_msg = String::new();
            let chat_messages: Vec<Value> = messages
                .iter()
                .filter(|m| {
                    if m.role == "system" {
                        sys_msg = m.content.clone();
                        false
                    } else {
                        true
                    }
                })
                .map(|m| {
                    serde_json::json!({
                        "role": m.role,
                        "content": m.content
                    })
                })
                .collect();

            let mut body = serde_json::json!({
                "model": agent.model,
                "messages": chat_messages,
                "max_tokens": 4096,
                "temperature": agent.temperature.unwrap_or(0.7),
            });

            if !sys_msg.is_empty() {
                body["system"] = serde_json::json!(sys_msg);
            }

            if let Some(sp) = &agent.system_prompt {
                body["system"] = serde_json::json!(sp);
            }

            let req_builder = client
                .post(format!("{}/messages", base_url))
                .json(&body)
                .header("x-api-key", agent.api_key.as_deref().unwrap_or(""))
                .header("anthropic-version", "2023-06-01");

            req_builder.build().map_err(|e: reqwest::Error| e.to_string())
        }
        "google" => {
            let chat_messages: Vec<Value> = messages
                .iter()
                .map(|m| {
                    serde_json::json!({
                        "role": m.role,
                        "parts": [{"text": m.content}]
                    })
                })
                .collect();

            let body = serde_json::json!({
                "contents": chat_messages,
                "generationConfig": {
                    "temperature": agent.temperature.unwrap_or(0.7),
                    "maxOutputTokens": 4096
                }
            });

            let api_key = agent.api_key.as_deref().unwrap_or("");
            let req_builder = client
                .post(format!(
                    "{}/models/{}:generateContent?key={}",
                    base_url, agent.model, api_key
                ))
                .json(&body);

            req_builder.build().map_err(|e: reqwest::Error| e.to_string())
        }
        _ => {
            let chat_messages: Vec<Value> = messages
                .iter()
                .map(|m| {
                    serde_json::json!({
                        "role": m.role,
                        "content": m.content
                    })
                })
                .collect();

            let body = serde_json::json!({
                "model": agent.model,
                "messages": chat_messages,
                "temperature": agent.temperature.unwrap_or(0.7),
                "max_tokens": 4096,
            });

            let mut req_builder =
                client.post(format!("{}/chat/completions", base_url)).json(&body);

            if let Some(key) = &agent.api_key {
                req_builder = req_builder.header("Authorization", format!("Bearer {}", key));
            }

            req_builder.build().map_err(|e: reqwest::Error| e.to_string())
        }
    }
}

#[tauri::command]
pub fn ai_chat(
    state: tauri::State<'_, AiManager>,
    agent_id: String,
    messages: Vec<ChatMessage>,
) -> Result<AgentResponse, String> {
    let agent = state.get_agent(&agent_id).ok_or("Agent not found")?;

    let req = build_request(&agent, &messages)?;
    let client = reqwest::blocking::Client::new();
    let resp = client.execute(req).map_err(|e: reqwest::Error| e.to_string())?;

    let status = resp.status();
    let body: Value = resp.json().map_err(|e: reqwest::Error| e.to_string())?;

    if !status.is_success() {
        return Err(format!("AI request failed ({}): {}", status, body));
    }

    let (content, input_tokens, output_tokens) = match agent.provider.as_str() {
        "anthropic" => (
            body["content"][0]["text"].as_str().unwrap_or("").to_string(),
            body["usage"]["input_tokens"].as_u64().unwrap_or(0),
            body["usage"]["output_tokens"].as_u64().unwrap_or(0),
        ),
        "google" => (
            body["candidates"][0]["content"]["parts"][0]["text"]
                .as_str()
                .unwrap_or("")
                .to_string(),
            body["usageMetadata"]["promptTokenCount"].as_u64().unwrap_or(0),
            body["usageMetadata"]["candidatesTokenCount"].as_u64().unwrap_or(0),
        ),
        _ => (
            body["choices"][0]["message"]["content"]
                .as_str()
                .unwrap_or("")
                .to_string(),
            body["usage"]["prompt_tokens"].as_u64().unwrap_or(0),
            body["usage"]["completion_tokens"].as_u64().unwrap_or(0),
        ),
    };

    let (price_in, price_out) = model_price(&agent.model);
    let cost = (input_tokens as f64 * price_in + output_tokens as f64 * price_out) / 1000.0;
    state.record_usage(&agent.id, input_tokens, output_tokens, cost);

    Ok(AgentResponse {
        agent_id: agent.id.clone(),
        content,
        provider: agent.provider.clone(),
        model: agent.model.clone(),
    })
}

#[tauri::command]
pub fn ai_get_usage(state: tauri::State<'_, AiManager>) -> Vec<AgentUsage> {
    state.get_usage()
}

#[tauri::command]
pub fn ai_reset_usage(state: tauri::State<'_, AiManager>) -> Result<(), String> {
    state.reset_usage();
    Ok(())
}

#[tauri::command]
pub fn ai_auto_route(
    state: tauri::State<'_, AiManager>,
    task_type: String,
) -> Vec<AgentConfig> {
    state.auto_route(&task_type)
}

#[tauri::command]
pub fn ai_list_agents(state: tauri::State<'_, AiManager>) -> Vec<AgentConfig> {
    state.list_agents()
}

#[tauri::command]
pub fn ai_update_agent(
    state: tauri::State<'_, AiManager>,
    config: AgentConfig,
) -> Result<(), String> {
    state.update_agent(config)
}

#[tauri::command]
pub fn ai_remove_agent(
    state: tauri::State<'_, AiManager>,
    agent_id: String,
) -> Result<(), String> {
    state.remove_agent(&agent_id)
}

#[derive(Serialize, Deserialize, Debug)]
pub struct AgentValidation {
    pub agent_id: String,
    pub valid: bool,
    pub error: Option<String>,
}

#[tauri::command]
pub fn ai_validate_agent(
    state: tauri::State<'_, AiManager>,
    agent_id: String,
) -> Result<AgentValidation, String> {
    let agent = state.get_agent(&agent_id).ok_or("Agent not found")?;

    if agent.api_key.is_none() || agent.api_key.as_deref().unwrap_or("").is_empty() {
        return Ok(AgentValidation {
            agent_id: agent.id.clone(),
            valid: false,
            error: Some("No API key configured".to_string()),
        });
    }

    let messages = vec![ChatMessage {
        role: "user".to_string(),
        content: "Say 'ok' and nothing else.".to_string(),
    }];

    let req = match build_request(&agent, &messages) {
        Ok(r) => r,
        Err(e) => {
            return Ok(AgentValidation {
                agent_id: agent.id.clone(),
                valid: false,
                error: Some(format!("Build request failed: {}", e)),
            });
        }
    };

    let client = match reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(15))
        .build()
    {
        Ok(c) => c,
        Err(e) => {
            return Ok(AgentValidation {
                agent_id: agent.id.clone(),
                valid: false,
                error: Some(format!("Client build failed: {}", e)),
            });
        }
    };

    match client.execute(req) {
        Ok(resp) => {
            let status = resp.status();
            if status.is_success() {
                Ok(AgentValidation {
                    agent_id: agent.id.clone(),
                    valid: true,
                    error: None,
                })
            } else {
                Ok(AgentValidation {
                    agent_id: agent.id.clone(),
                    valid: false,
                    error: Some(format!("API error ({}): check your API key and model", status)),
                })
            }
        }
        Err(e) => {
            Ok(AgentValidation {
                agent_id: agent.id.clone(),
                valid: false,
                error: Some(format!("Connection failed: {}", e)),
            })
        }
    }
}

#[tauri::command]
pub fn ai_set_provider(
    state: tauri::State<'_, AiManager>,
    provider: String,
    base_url: Option<String>,
    api_key: Option<String>,
) -> Result<(), String> {
    state.set_provider(&provider, base_url, api_key)
}
