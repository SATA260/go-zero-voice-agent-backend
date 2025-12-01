
from typing import List
from fastapi import APIRouter, Body, Form, HTTPException, Query, Request, UploadFile


router = APIRouter()

def get_user_id(request: Request) -> str:
    user_id = request.headers.get("X-User-ID")
    if not user_id:
        raise HTTPException(status_code=401, detail="X-User-ID header missing")
    return user_id

@router.get("/health")
async def health_check():
    return {"status": "ok"}

@router.get("/documents")
async def get_documents_by_ids(request: Request, ids: List[str] = Query(...)):
    # Example implementation, replace with actual data retrieval logic
    documents = [{"id": doc_id, "content": f"Content for document {doc_id}"} for doc_id in ids]
    return {"documents": documents}

@router.delete("/documents")
async def delete_documents(request: Request, ids: List[str] = Body(...)):
    # Example implementation, replace with actual data deletion logic
    deleted_count = len(ids)  # Assume all documents are deleted successfully
    return {"deleted_count": deleted_count}

@router.post("/embed")
async def embed_file_upload(
    request: Request,
    file_id: str = Form(...),
    file_path: str = Form(...),
    entity_id: str = Form(None),
):
    user_id = get_user_id(request)
    return {"message": f"File {file_id} at {file_path} embedded successfully for user {user_id}."}