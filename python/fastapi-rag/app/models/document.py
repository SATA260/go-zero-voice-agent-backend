from enum import Enum
import hashlib
from pydantic import BaseModel
from typing import Optional

class DocumentResponse(BaseModel):
    page_content: str
    metadata: dict

class DocumentModel(BaseModel):
    page_content: str
    metadata: Optional[dict] = None
    def generate_digest(self):
        hash_obj = hashlib.md5(self.page_content.encode())
        return hash_obj.hexdigest()
    
class StoreDocument(BaseModel):
    filepath: str
    filename: str
    file_content_type: str
    file_id: str

class QueryRequestBody(BaseModel):
    query: str
    file_id: str
    top_k: int = 4
    entity_id: Optional[str] = None

class CleanupMethod(str, Enum):
    incremental = "incremental"
    full = "full"

class QueryMultipleBody(BaseModel):
    query: str
    file_ids: list[str]
    top_k: int = 4