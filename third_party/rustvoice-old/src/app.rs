use crate::{
    callrecord::{CallRecordManagerBuilder, CallRecordSender},
    config::Config,
    handler::call::ActiveCallRef,
    media::engine::StreamEngine,
    // proxy imports removed - only WebRTC and WebSocket calls supported
    // useragent imports removed - only WebRTC and WebSocket calls supported
};
use anyhow::Result;
use axum::Router;
use futures::lock::Mutex;
use std::path::Path;
use std::sync::Arc;
use std::{collections::HashMap, net::SocketAddr};
use tokio::net::TcpListener;
use tokio::select;
use tokio_util::sync::CancellationToken;
use tower_http::cors::{AllowOrigin, CorsLayer};
use tracing::{info, warn};

pub struct AppStateInner {
    pub config: Arc<Config>,
    // useragent removed - only WebRTC and WebSocket calls supported
    pub token: CancellationToken,
    pub active_calls: Arc<Mutex<HashMap<String, ActiveCallRef>>>,
    pub stream_engine: Arc<StreamEngine>,
    pub callrecord_sender: tokio::sync::Mutex<Option<CallRecordSender>>,
}

pub type AppState = Arc<AppStateInner>;

pub struct AppStateBuilder {
    pub config: Option<Config>,
    // useragent removed - only WebRTC and WebSocket calls supported
    pub stream_engine: Option<Arc<StreamEngine>>,
    pub callrecord_sender: Option<CallRecordSender>,
    pub cancel_token: Option<CancellationToken>,
}

impl AppStateInner {
    pub fn get_dump_events_file(&self, session_id: &String) -> String {
        let root = Path::new(&self.config.recorder_path);
        root.join(format!("{}.events.jsonl", session_id))
            .to_string_lossy()
            .to_string()
    }

    pub fn get_recorder_file(&self, session_id: &String) -> String {
        let root = Path::new(&self.config.recorder_path);
        if !root.exists() {
            match std::fs::create_dir_all(root) {
                Ok(_) => {
                    info!("created recorder root: {}", root.to_string_lossy());
                }
                Err(e) => {
                    warn!(
                        "Failed to create recorder root: {} {}",
                        e,
                        root.to_string_lossy()
                    );
                }
            }
        }
        root.join(session_id)
            .with_extension("wav")
            .to_string_lossy()
            .to_string()
    }
}

impl AppStateBuilder {
    pub fn new() -> Self {
        Self {
            config: None,
            // useragent removed - only WebRTC and WebSocket calls supported
            stream_engine: None,
            callrecord_sender: None,
            cancel_token: None,
        }
    }

    pub fn config(mut self, config: Config) -> Self {
        self.config = Some(config);
        self
    }

    // with_useragent method removed - only WebRTC and WebSocket calls supported

    pub fn with_stream_engine(mut self, stream_engine: Arc<StreamEngine>) -> Self {
        self.stream_engine = Some(stream_engine);
        self
    }

    pub fn with_callrecord_sender(mut self, sender: CallRecordSender) -> Self {
        self.callrecord_sender = Some(sender);
        self
    }

    pub fn with_cancel_token(mut self, token: CancellationToken) -> Self {
        self.cancel_token = Some(token);
        self
    }

    pub async fn build(self) -> Result<AppState> {
        let config: Arc<Config> = Arc::new(self.config.unwrap_or_default());
        let token = self
            .cancel_token
            .unwrap_or_else(|| CancellationToken::new());
        let _ = crate::media::cache::set_cache_dir(&config.media_cache_path);

        // useragent creation removed - only WebRTC and WebSocket calls supported
        let stream_engine = self.stream_engine.unwrap_or_default();

        Ok(Arc::new(AppStateInner {
            config,
            // useragent removed - only WebRTC and WebSocket calls supported
            token,
            active_calls: Arc::new(Mutex::new(HashMap::new())),
            stream_engine,
            callrecord_sender: tokio::sync::Mutex::new(self.callrecord_sender),
        }))
    }
}

// build_sip_server function removed - only WebRTC and WebSocket calls supported

pub async fn run(state: AppState) -> Result<()> {
    // useragent removed - only WebRTC and WebSocket calls supported
    let token = state.token.clone();

    let router = create_router(state.clone());
    let addr: SocketAddr = state.config.http_addr.parse()?;
    let listener = match TcpListener::bind(addr).await {
        Ok(l) => l,
        Err(e) => {
            tracing::error!("Failed to bind to {}: {}", addr, e);
            return Err(anyhow::anyhow!("Failed to bind to {}: {}", addr, e));
        }
    };

    if let Some(ref callrecord) = state.config.callrecord {
        let mut callrecord_sender = state.callrecord_sender.lock().await;
        if callrecord_sender.is_none() {
            let mut callrecord_manager = CallRecordManagerBuilder::new()
                .with_cancel_token(token.child_token())
                .with_config(callrecord.clone())
                .build();
            callrecord_sender.replace(callrecord_manager.sender.clone());

            tokio::spawn(async move {
                callrecord_manager.serve().await;
            });
        }
    }

    // proxy server removed - only WebRTC and WebSocket calls supported

    let http_task = axum::serve(
        listener,
        router.into_make_service_with_connect_info::<SocketAddr>(),
    );
    select! {
        http_result = http_task => {
            match http_result {
                Ok(_) => info!("Server shut down gracefully"),
                Err(e) => {
                    tracing::error!("Server error: {}", e);
                    return Err(anyhow::anyhow!("Server error: {}", e));
                }
            }
        }
        // useragent serve removed - only WebRTC and WebSocket calls supported
        _ = token.cancelled() => {
            info!("Application shutting down due to cancellation");
        }
    }
    token.cancel();
    // useragent stop removed - only WebRTC and WebSocket calls supported
    Ok(())
}

// Static file handlers removed - only API endpoints supported

fn create_router(state: AppState) -> Router {
    // CORS configuration to allow cross-origin requests
    let cors = CorsLayer::new()
        .allow_origin(AllowOrigin::any())
        .allow_methods([
            axum::http::Method::GET,
            axum::http::Method::POST,
            axum::http::Method::PUT,
            axum::http::Method::DELETE,
            axum::http::Method::OPTIONS,
        ])
        .allow_headers([
            axum::http::header::CONTENT_TYPE,
            axum::http::header::AUTHORIZATION,
            axum::http::header::ACCEPT,
            axum::http::header::ORIGIN,
        ]);

    // Only API routes - no static file serving
    let call_routes = crate::handler::router().with_state(state);

    Router::new()
        .merge(call_routes)
        .layer(cors)
}
