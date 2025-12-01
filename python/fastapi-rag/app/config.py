
import datetime
from enum import Enum
import json
import logging
import os
from urllib.parse import quote_plus
from dotenv import load_dotenv, find_dotenv
from starlette.middleware.base import BaseHTTPMiddleware
from langchain_openai import OpenAIEmbeddings

from app.db.vector_store.factor import get_vector_store



load_dotenv(find_dotenv())

class LogLevel(Enum):
    FATAL = "fatal"
    ERROR = "error"
    WARN = "warn"
    INFO = "info"
    DEBUG = "debug"
    NOTSET = "notset"

class VectorDBType(Enum):
    PGVECTOR = "pgvector"

class EmbeddingProvider(Enum):
    OPENAI = "openai"

def get_env_variable(
    var_name: str, default_value: str = None, required: bool = False
) -> str:
    value = os.getenv(var_name)
    if value is None:
        if default_value is None and required:
            raise ValueError(f"Environment variable '{var_name}' not found.")
        return default_value
    return value

# VECTOR DB CONFIGURATION
VECTOR_DB_TYPE = VectorDBType(
    get_env_variable("VECTOR_DB_TYPE", VectorDBType.PGVECTOR.value)
)
POSTGRES_DB = get_env_variable("POSTGRES_DB")
POSTGRES_USER = get_env_variable("POSTGRES_USER")
POSTGRES_PASSWORD = get_env_variable("POSTGRES_PASSWORD")
DB_HOST = get_env_variable("DB_HOST", "localhost")
DB_PORT = get_env_variable("DB_PORT", "5432")
COLLECTION_NAME = get_env_variable("COLLECTION_NAME", "documents")

CHUNK_SIZE = int(get_env_variable("CHUNK_SIZE", "1500"))
CHUNK_OVERLAP = int(get_env_variable("CHUNK_OVERLAP", "100"))

pg_connection_suffix = f"{quote_plus(POSTGRES_USER)}:{quote_plus(POSTGRES_PASSWORD)}@{DB_HOST}:{DB_PORT}/{quote_plus(POSTGRES_DB)}"
PG_CONNECTION_STRING = f"postgresql+psycopg2://{pg_connection_suffix}"
PG_DSN = f"postgresql://{pg_connection_suffix}"

RAG_HOST = get_env_variable("RAG_HOST", "localhost", True)
RAG_PORT = int(get_env_variable("RAG_PORT", "8000", True))

# Logging Configuration

HTTP_REQ = "http_req"
HTTP_RESP = "http_resp"

logger = logging.getLogger()

LOGGING_LEVEL = get_env_variable("LOGGING_LEVEL", LogLevel.INFO.value)
if LOGGING_LEVEL == LogLevel.DEBUG.value:
    logger.setLevel(logging.DEBUG)
elif LOGGING_LEVEL == LogLevel.INFO.value:
    logger.setLevel(logging.INFO)
elif LOGGING_LEVEL == LogLevel.WARN.value:
    logger.setLevel(logging.WARN)
elif LOGGING_LEVEL == LogLevel.ERROR.value:
    logger.setLevel(logging.ERROR)
elif LOGGING_LEVEL == LogLevel.FATAL.value:
    logger.setLevel(logging.FATAL)
else:
    logger.setLevel(logging.INFO)

CONSOLE_JSON = get_env_variable("CONSOLE_JSON", "false").lower() == "true"
if CONSOLE_JSON:
    
    class JsonFormatter(logging.Formatter):
        def __init__(self):
            super(JsonFormatter, self).__init__()
        def format(self, record):
            json_record = {}

            json_record["message"] = record.getMessage()

            if HTTP_REQ in record.__dict__:
                json_record[HTTP_REQ] = record.__dict__[HTTP_REQ]

            if HTTP_RESP in record.__dict__:
                json_record[HTTP_RESP] = record.__dict__[HTTP_RESP]

            if record.levelno == logging.ERROR and record.exc_info:
                json_record["exception"] = self.formatException(record.exc_info)

            timestamp = datetime.fromtimestamp(record.created)
            json_record["timestamp"] = timestamp.isoformat()

            # add level
            json_record["level"] = record.levelname
            json_record["filename"] = record.filename
            json_record["lineno"] = record.lineno
            json_record["funcName"] = record.funcName
            json_record["module"] = record.module
            json_record["threadName"] = record.threadName

            return json.dumps(json_record)
        
    formatter = JsonFormatter()
else:
    formatter = logging.Formatter(
        "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    )

handler = logging.StreamHandler()
handler.setFormatter(formatter)
logger.addHandler(handler)

class LogMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request, call_next):
        response = await call_next(request)

        logger_method = logger.info

        if str(request.url).endswith("/health"):
            logger_method = logger.debug

        logger_method(
            f"Request {request.method} {request.url} - {response.status_code}",
            extra={
                HTTP_REQ: {"method": request.method, "url": str(request.url)},
                HTTP_RESP: {"status_code": response.status_code},
            },
        )

        return response


logging.getLogger("uvicorn.access").disabled = True

# Credentials and API Keys
RAG_OPENAI_API_KEY = get_env_variable("RAG_OPENAI_API_KEY", None)
RAG_OPENAI_BASEURL = get_env_variable("RAG_OPENAI_BASEURL", None)
RAG_OPENAI_MODEL = get_env_variable("RAG_OPENAI_MODEL", None)
EMBEDDING_CHUNK_SIZE = int(get_env_variable("EMBEDDING_CHUNK_SIZE", "200"))
RAG_CHECK_EMBEDDING_CTX_LENGTH = True if get_env_variable("RAG_CHECK_EMBEDDING_CTX_LENGTH", "true").lower() == "true" else False
# Embedding Configuration
embeddings = OpenAIEmbeddings(
    model = RAG_OPENAI_MODEL,
    api_key = RAG_OPENAI_API_KEY,
    openai_api_base = RAG_OPENAI_BASEURL,
    chunk_size= EMBEDDING_CHUNK_SIZE,
    check_embedding_ctx_length=RAG_CHECK_EMBEDDING_CTX_LENGTH
)
logger.info(f"Initialized embeddings of type: {type(embeddings)}")

# Vector store configuration
if VECTOR_DB_TYPE == VectorDBType.PGVECTOR:
    vector_store = get_vector_store(
        connection_string=PG_CONNECTION_STRING,
        embeddings=embeddings,
        collection_name=COLLECTION_NAME,
        mode="async",
    )
else:
    raise ValueError(f"Unsupported VECTOR_DB_TYPE: {VECTOR_DB_TYPE}")

retriever = vector_store.as_retriever()

known_source_ext = [
    "go",
    "py",
    "java",
    "sh",
    "bat",
    "ps1",
    "cmd",
    "js",
    "ts",
    "css",
    "cpp",
    "hpp",
    "h",
    "c",
    "cs",
    "sql",
    "log",
    "ini",
    "pl",
    "pm",
    "r",
    "dart",
    "dockerfile",
    "env",
    "php",
    "hs",
    "hsc",
    "lua",
    "nginxconf",
    "conf",
    "m",
    "mm",
    "plsql",
    "perl",
    "rb",
    "rs",
    "db2",
    "scala",
    "bash",
    "swift",
    "vue",
    "svelte",
    "yml",
    "yaml",
    "eml",
    "ex",
    "exs",
    "erl",
    "tsx",
    "jsx",
    "lhs",
]