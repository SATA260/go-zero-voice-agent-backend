# Go-Zero 应用日志配置指南

## 配置说明

Go-Zero 的 `rest.RestConf` 和 `zrpc.RpcServerConf` 都内置了日志配置（`LogConf`）。你只需要在配置文件中添加 `Log` 字段即可。

## API 服务配置示例

### chatroom API 配置 (chatroom.yaml)

```yaml
Name: chatroom
Host: ${CHATROOM_API_HOST}
Port: ${CHATROOM_API_PORT}

# 日志配置 - 添加此部分
Log:
  ServiceName: chatroom-api
  Mode: file              # file: 输出到文件, console: 输出到控制台
  Path: logs              # 日志文件路径，相对于项目根目录
  Level: info             # debug, info, error, severe
  Encoding: json          # json 或 plain (Filebeat 需要 json 格式)
  KeepDays: 7             # 日志文件保留天数
  Compress: true          # 是否压缩旧日志文件
  MaxBackups: 5           # 最多保留的备份文件数
  MaxSize: 100            # 单个日志文件最大大小 (MB)

ChatroomRpcConf:
  Etcd:
    Hosts:
    - ${ETCD_HOST}
    Key: chatroom.rpc

Websocket:
  MaxConnections: ${WS_MAX_CONNECTIONS}
  # ... 其他配置
```

### usercenter API 配置 (usercenter.yaml)

```yaml
Name: usercenter
Host: ${USERCENTER_API_HOST}
Port: ${USERCENTER_API_PORT}

# 日志配置
Log:
  ServiceName: usercenter-api
  Mode: file
  Path: logs
  Level: info
  Encoding: json
  KeepDays: 7

UsercenterRpcConf:
  Etcd:
    Hosts:
    - ${ETCD_HOST}
    Key: usercenter.rpc
```

### voicechat API 配置 (voicechat.yaml)

```yaml
Name: voiceChat
Host: ${VOICECHAT_API_HOST}
Port: ${VOICECHAT_API_PORT}

# 日志配置
Log:
  ServiceName: voicechat-api
  Mode: file
  Path: logs
  Level: info
  Encoding: json
  KeepDays: 7

LlmRpcConf:
  Etcd:
    Hosts:
    - ${ETCD_HOST}
    Key: llmservice.rpc

VoiceChatRpcConf:
  Etcd:
    Hosts:
    - ${ETCD_HOST}
    Key: voicechat.rpc

RustPBXConfig:
  Url: ${RUST_PBX_URL}
  WebSocketUrl: ${RUST_PBX_WEBSOCKET_CALL_URL}
```

## RPC 服务配置示例

RPC 服务的日志配置与 API 服务相同。以 usercenter RPC 为例：

```yaml
Name: usercenter.rpc
ListenOn: ${USERCENTER_RPC_HOST}:${USERCENTER_RPC_PORT}

# 日志配置
Log:
  ServiceName: usercenter-rpc
  Mode: file
  Path: logs
  Level: info
  Encoding: json
  KeepDays: 7

Etcd:
  Hosts:
  - ${ETCD_HOST}
  Key: usercenter.rpc

# ... 其他配置
```

## 日志路径说明

### 相对路径（推荐）

```yaml
Log:
  Path: logs  # 相对于项目根目录
```

日志文件将输出到：
- `/home/zhang/go-zero-voice-agent/go-zero-voice-agent-backend/logs/`

### 绝对路径

```yaml
Log:
  Path: /home/zhang/go-zero-voice-agent/go-zero-voice-agent-backend/logs
```

## 日志文件命名规则

Go-Zero 会自动按照服务名称和日期创建日志文件：

```
logs/
├── access.log              # API 访问日志
├── error.log               # 错误日志
├── severe.log              # 严重错误日志
├── stat.log                # 统计日志
└── slow.log                # 慢请求日志
```

如果配置了 `ServiceName`，文件名会包含服务名：

```
logs/
├── chatroom-api-access.log
├── chatroom-api-error.log
├── chatroom-api-severe.log
└── chatroom-api-stat.log
```

## 日志级别说明

- `debug`: 调试信息，包含最详细的日志
- `info`: 一般信息，包括请求处理、业务流程等（推荐）
- `error`: 错误信息，包括异常和错误
- `severe`: 严重错误，需要立即关注的问题

## 开发环境 vs 生产环境

### 开发环境

```yaml
Log:
  Mode: console    # 输出到控制台，方便调试
  Level: debug     # 显示详细日志
  Encoding: plain  # 纯文本格式，易读
```

### 生产环境（配合 Filebeat）

```yaml
Log:
  Mode: file       # 输出到文件
  Level: info      # 只记录重要信息
  Encoding: json   # JSON 格式，便于结构化日志收集
  KeepDays: 7      # 自动清理旧日志
  Compress: true   # 压缩旧日志节省空间
```

## 日志字段说明

使用 JSON 格式时，日志包含以下字段：

```json
{
  "@timestamp": "2025-12-27T17:30:00.123+08:00",
  "level": "info",
  "content": "用户登录成功",
  "caller": "handler/login.go:45",
  "span": "trace-id-123",
  "trace": "span-id-456",
  "duration": "ms:15"
}
```

## 快速配置脚本

你可以使用以下命令批量为所有服务添加日志配置：

```bash
# 为所有 API 服务添加日志配置
for yaml_file in app/*/cmd/api/etc/*.yaml; do
  echo "配置: $yaml_file"
  # 手动编辑或使用脚本添加 Log 配置
done

# 为所有 RPC 服务添加日志配置
for yaml_file in app/*/cmd/rpc/etc/*.yaml; do
  echo "配置: $yaml_file"
  # 手动编辑或使用脚本添加 Log 配置
done
```

## 验证日志输出

1. **启动服务**：
```bash
cd app/usercenter/cmd/api
go run usercenter.go -f etc/usercenter.yaml
```

2. **检查日志文件**：
```bash
cd /home/zhang/go-zero-voice-agent/go-zero-voice-agent-backend
ls -la logs/
tail -f logs/usercenter-api-access.log
```

3. **测试 API 并查看日志**：
```bash
# 发送测试请求
curl http://localhost:8888/api/test

# 实时查看日志
tail -f logs/usercenter-api-access.log
```

## 注意事项

1. **确保日志目录存在**：服务启动前需要创建 `logs` 目录
2. **权限问题**：确保应用有写入日志目录的权限
3. **磁盘空间**：定期清理或设置合理的 `KeepDays` 避免磁盘满
4. **性能影响**：大量日志会影响性能，生产环境使用 `info` 级别
5. **JSON 格式**：Filebeat 需要 JSON 格式才能正确解析日志

## 与 Filebeat 集成

配置完成后，Filebeat 会自动收集 `logs/` 目录下的所有日志文件并发送到 Kafka，然后通过 go-stash 写入 Elasticsearch。

确保：
1. ✅ 日志格式为 `json`
2. ✅ 日志路径为 `logs` （相对路径）
3. ✅ Filebeat 配置的路径与实际日志路径一致
