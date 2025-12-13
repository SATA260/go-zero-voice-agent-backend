package consts

const (
	TOOL_CALLING_START                = "tool_calling_start"
	TOOL_CALLING_EXECUTING            = "tool_calling_executing"
	TOOL_CALLING_FINISHED             = "tool_calling_finished"
	TOOL_CALLING_WAITING_CONFIRMATION = "tool_calling_waiting_confirmation"
	TOOL_CALLING_CONFIRMED            = "tool_calling_confirmed"
	TOOL_CALLING_REJECTED             = "tool_calling_rejected"
	TOOL_CALLING_FAILED               = "tool_calling_failed"
)

const (
	TOOL_CALLING_SELF_RAG = "self_rag"
)

const (
	TOOL_CALLING_SCOPE_SERVER = "server"
	TOOL_CALLING_SCOPE_CLIENT = "client"
)