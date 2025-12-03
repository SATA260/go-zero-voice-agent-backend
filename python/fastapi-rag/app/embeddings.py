"""Custom embedding client that wraps DashScope's OpenAI-compatible API."""

from __future__ import annotations

from typing import Iterable, List, Optional, Sequence

from langchain_core.embeddings import Embeddings
from openai import AsyncOpenAI, OpenAI


class DashScopeEmbeddings(Embeddings):
    """LangChain-compatible embeddings powered by the DashScope OpenAI API."""

    def __init__(
        self,
        *,
        api_key: str,
        base_url: str,
        model: str,
        chunk_size: int = 200,
        dimensions: Optional[int] = None,
        encoding_format: str = "float",
    ) -> None:
        if not api_key:
            raise ValueError("DashScope API key must be provided")
        if not base_url:
            raise ValueError("DashScope base URL must be provided")
        if not model:
            raise ValueError("Embedding model name must be provided")
        self._client = OpenAI(api_key=api_key, base_url=base_url)
        self._async_client = AsyncOpenAI(api_key=api_key, base_url=base_url)
        self._model = model
        self._chunk_size = max(1, chunk_size)
        self._dimensions = dimensions
        self._encoding_format = encoding_format

    def _chunk_texts(self, texts: Sequence[str]) -> Iterable[Sequence[str]]:
        for index in range(0, len(texts), self._chunk_size):
            yield texts[index : index + self._chunk_size]

    @staticmethod
    def _prepare_inputs(texts: Sequence[str]) -> List[str]:
        return [text.replace("\n", " ") for text in texts]

    def _request_embeddings(self, payload: Sequence[str]) -> List[List[float]]:
        params = {
            "model": self._model,
            "input": list(payload),
            "encoding_format": self._encoding_format,
        }
        if self._dimensions is not None:
            params["dimensions"] = self._dimensions
        response = self._client.embeddings.create(**params)
        # API preserves ordering via the index attribute.
        return [record.embedding for record in sorted(response.data, key=lambda item: item.index)]

    async def _arequest_embeddings(self, payload: Sequence[str]) -> List[List[float]]:
        params = {
            "model": self._model,
            "input": list(payload),
            "encoding_format": self._encoding_format,
        }
        if self._dimensions is not None:
            params["dimensions"] = self._dimensions
        response = await self._async_client.embeddings.create(**params)
        return [record.embedding for record in sorted(response.data, key=lambda item: item.index)]

    def embed_documents(self, texts: List[str]) -> List[List[float]]:
        inputs = self._prepare_inputs(texts)
        embeddings: List[List[float]] = []
        for batch in self._chunk_texts(inputs):
            embeddings.extend(self._request_embeddings(batch))
        return embeddings

    async def aembed_documents(self, texts: List[str]) -> List[List[float]]:
        inputs = self._prepare_inputs(texts)
        embeddings: List[List[float]] = []
        for batch in self._chunk_texts(inputs):
            embeddings.extend(await self._arequest_embeddings(batch))
        return embeddings

    def embed_query(self, text: str) -> List[float]:
        return self.embed_documents([text])[0]

    async def aembed_query(self, text: str) -> List[float]:
        results = await self.aembed_documents([text])
        return results[0]
