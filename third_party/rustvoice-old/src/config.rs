// proxy imports removed - only WebRTC and WebSocket calls supported
use anyhow::Error;
use clap::Parser;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

// USER_AGENT constant removed - only WebRTC and WebSocket calls supported

#[derive(Parser, Debug)]
#[command(version)]
pub(crate) struct Cli {
    #[clap(long, default_value = "rustpbx.toml")]
    pub conf: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Config {
    pub http_addr: String,
    pub log_level: Option<String>,
    pub log_file: Option<String>,
    pub console: Option<ConsoleConfig>,
    // ua (useragent) removed - only WebRTC and WebSocket calls supported
    // proxy removed - only WebRTC and WebSocket calls supported
    pub recorder_path: String,
    pub callrecord: Option<CallRecordConfig>,
    pub media_cache_path: String,
    pub llmproxy: Option<String>,
    pub ice_servers: Option<Vec<IceServerItem>>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ConsoleConfig {
    pub prefix: String,
    pub username: Option<String>,
    pub password: Option<String>,
}

#[derive(Debug, Deserialize, Default, Serialize)]
pub struct IceServerItem {
    pub urls: Vec<String>,
    pub username: Option<String>,
    pub password: Option<String>,
}

impl Default for ConsoleConfig {
    fn default() -> Self {
        Self {
            prefix: "/console".to_string(),
            username: None,
            password: None,
        }
    }
}

// UseragentConfig and InviteHandlerConfig removed - only WebRTC and WebSocket calls supported

// UserBackendConfig and LocatorConfig removed - only WebRTC and WebSocket calls supported

#[derive(Debug, Deserialize, Clone, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum S3Vendor {
    Aliyun,
    Tencent,
    Minio,
    AWS,
    GCP,
    Azure,
    DigitalOcean,
}

#[derive(Debug, Deserialize, Clone, Serialize)]
#[serde(tag = "type")]
#[serde(rename_all = "snake_case")]
pub enum CallRecordConfig {
    Local {
        root: String,
    },
    S3 {
        vendor: S3Vendor,
        bucket: String,
        region: String,
        access_key: String,
        secret_key: String,
        endpoint: String,
        root: String,
        with_media: Option<bool>,
    },
    Http {
        url: String,
        headers: Option<HashMap<String, String>>,
        with_media: Option<bool>,
    },
}

// MediaProxyMode and MediaProxyConfig removed - only WebRTC and WebSocket calls supported

// ProxyConfig removed - only WebRTC and WebSocket calls supported

// UserBackendConfig and LocatorConfig Default implementations removed - only WebRTC and WebSocket calls supported

// UseragentConfig Default implementation removed - only WebRTC and WebSocket calls supported

impl Default for CallRecordConfig {
    fn default() -> Self {
        Self::Local {
            #[cfg(target_os = "windows")]
            root: "./cdr".to_string(),
            #[cfg(not(target_os = "windows"))]
            root: "/tmp/cdr".to_string(),
        }
    }
}

impl Default for Config {
    fn default() -> Self {
        Self {
            http_addr: "0.0.0.0:8080".to_string(),
            log_level: Some("info".to_string()),
            log_file: None,
            console: Some(ConsoleConfig::default()),
            // ua (useragent) removed - only WebRTC and WebSocket calls supported
            // proxy removed - only WebRTC and WebSocket calls supported
            #[cfg(target_os = "windows")]
            recorder_path: "./recorder".to_string(),
            #[cfg(not(target_os = "windows"))]
            recorder_path: "/tmp/recorder".to_string(),
            #[cfg(target_os = "windows")]
            media_cache_path: "./mediacache".to_string(),
            #[cfg(not(target_os = "windows"))]
            media_cache_path: "/tmp/mediacache".to_string(),
            callrecord: None,
            llmproxy: None,
            ice_servers: None,
        }
    }
}

impl Config {
    pub fn load(path: &str) -> Result<Self, Error> {
        let config = toml::from_str(
            &std::fs::read_to_string(path).map_err(|e| anyhow::anyhow!("{}: {}", e, path))?,
        )?;
        Ok(config)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config_load() {
        let config = Config::default();
        let config_str = toml::to_string(&config).unwrap();
        println!("{}", config_str);
    }
    #[test]
    fn test_config_dump() {
        let config = Config::default();
        let config_str = toml::to_string(&config).unwrap();
        println!("{}", config_str);
    }
}
