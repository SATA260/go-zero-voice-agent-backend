# 日志配置完成汇总

## ✅ 已配置的服务

所有服务的日志配置已完成，统一使用以下配置：

### API 服务 (5个)

1. **chatroom-api** - [app/chatroom/cmd/api/etc/chatroom.yaml](../../../app/chatroom/cmd/api/etc/chatroom.yaml:5-13)
2. **usercenter-api** - [app/usercenter/cmd/api/etc/usercenter.yaml](../../../app/usercenter/cmd/api/etc/usercenter.yaml:5-13)
3. **voicechat-api** - [app/voicechat/cmd/api/etc/voicechat.yaml](../../../app/voicechat/cmd/api/etc/voicechat.yaml:5-13)
4. **llm-api** - [app/llm/cmd/api/etc/llm.yaml](../../../app/llm/cmd/api/etc/llm.yaml:5-13)
5. **rag-api** - [app/rag/cmd/api/etc/rag.yaml](../../../app/rag/cmd/api/etc/rag.yaml:5-13)

### RPC 服务 (4个)

1. **usercenter-rpc** - [app/usercenter/cmd/rpc/etc/usercenter.yaml](../../../app/usercenter/cmd/rpc/etc/usercenter.yaml:4-12)
2. **voicechat-rpc** - [app/voicechat/cmd/rpc/etc/voicechat.yaml](../../../app/voicechat/cmd/rpc/etc/voicechat.yaml:4-12)
3. **llm-rpc** - [app/llm/cmd/rpc/etc/llmservice.yaml](../../../app/llm/cmd/rpc/etc/llmservice.yaml:4-12)
4. **rag-rpc** - [app/rag/cmd/rpc/etc/rag.yaml](../../../app/rag/cmd/rpc/etc/rag.yaml:4-12)

### Job 服务 (1个)

1. **mqueue-job** - [app/mqueue/cmd/job/etc/mqueue.yaml](../../../app/mqueue/cmd/job/etc/mqueue.yaml:4-12)

## 📝 统一日志配置

```yaml
Log:
  ServiceName: <服务名称>  # 如: chatroom-api, usercenter-rpc
  Mode: file              # 输出到文件
  Path: logs              # 统一日志目录
  Level: info             # 日志级别
  Encoding: json          # JSON 格式（Filebeat 需要）
  KeepDays: 7             # 保留 7 天
  Compress: true          # 压缩旧日志
```

## 📂 日志文件结构

启动服务后，日志文件将按以下结构生成：

```
/home/zhang/go-zero-voice-agent/go-zero-voice-agent-backend/logs/
├── chatroom-api-access.log      # API 访问日志
├── chatroom-api-error.log       # 错误日志
├── chatroom-api-severe.log      # 严重错误日志
├── chatroom-api-stat.log        # 统计日志
├── usercenter-api-access.log
├── usercenter-api-error.log
├── usercenter-rpc-access.log
├── usercenter-rpc-error.log
├── voicechat-api-access.log
├── voicechat-rpc-access.log
├── llm-api-access.log
├── llm-rpc-access.log
├── rag-api-access.log
├── rag-rpc-access.log
├── mqueue-job-access.log
└── ...
```

## 🚀 下一步操作

### 1. 启动日志收集系统

使用自动化脚本启动完整的日志收集栈：

```bash
cd /home/zhang/go-zero-voice-agent/go-zero-voice-agent-backend
./deploy/filebeat/start-logging.sh
```

或者手动启动：

```bash
# 启动 Kafka
docker compose -f docker-compose-env.yml up -d kafka

# 等待 30 秒后创建 Topic
sleep 30
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create --bootstrap-server localhost:9092 \
  --topic looklook-log --partitions 3 --replication-factor 1

# 启动 Elasticsearch 和 Kibana
docker compose -f docker-compose-env.yml up -d elasticsearch kibana

# 启动 Filebeat 和 go-stash
docker compose -f docker-compose-env.yml up -d filebeat go-stash
```

### 2. 启动你的应用服务

启动任意服务，日志将自动输出到 `logs/` 目录：

```bash
cd app/usercenter/cmd/api
go run usercenter.go -f etc/usercenter.yaml
```

### 3. 验证日志生成

```bash
# 查看日志文件
ls -lh logs/

# 实时查看日志
tail -f logs/usercenter-api-access.log

# 查看 JSON 格式日志
cat logs/usercenter-api-access.log | jq .
```

### 4. 验证日志收集

```bash
# 查看 Filebeat 状态
docker logs -f filebeat

# 查看 Kafka 消息
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic looklook-log \
  --max-messages 5

# 查看 Elasticsearch 索引
curl http://localhost:9200/_cat/indices?v
```

### 5. 在 Kibana 查看日志

1. 访问 http://localhost:5601
2. 进入 **Management** → **Stack Management** → **Index Patterns**
3. 创建索引模式：`go-zero-voice-agent-*`
4. 选择时间字段：`@timestamp`
5. 在 **Discover** 中查看和搜索日志

## 🔍 日志字段说明

每条 JSON 日志包含以下字段：

```json
{
  "@timestamp": "2025-12-27T17:30:00.123+08:00",
  "level": "info",
  "content": "HTTP Request Log",
  "caller": "handler/handler.go:45",
  "span": "trace-id-123",
  "trace": "span-id-456",
  "duration": "ms:15",
  "method": "GET",
  "path": "/api/user/info",
  "status": 200
}
```

## 📊 Kibana 常用查询

在 Kibana 的 Discover 页面，可以使用以下查询：

```
# 查看错误日志
level: "error"

# 查看特定服务日志
log_source: "go-zero-voice-agent" AND content: *usercenter*

# 查看慢请求（超过 1 秒）
duration > 1000

# 查看特定 API 路径
path: "/api/user/login"

# 查看 HTTP 错误状态码
status >= 400
```

## 🛠 故障排查

### 日志文件未生成

1. 检查 `logs/` 目录是否存在：`ls -ld logs/`
2. 检查权限：`chmod 755 logs/`
3. 检查服务配置是否正确
4. 查看服务启动日志是否有错误

### Filebeat 无法收集日志

1. 检查日志文件是否为 JSON 格式：`cat logs/*.log | head -5`
2. 查看 Filebeat 日志：`docker logs filebeat`
3. 检查文件路径配置是否正确
4. 确认 Filebeat 容器有读取权限

### Kibana 查询不到日志

1. 检查 go-stash 状态：`docker logs go-stash`
2. 验证 Elasticsearch 索引：`curl http://localhost:9200/_cat/indices?v`
3. 检查 Kafka 中是否有消息
4. 确认索引模式配置正确

## 📚 相关文档

- [Filebeat 部署指南](README.md)
- [日志配置详细说明](LOG_CONFIG_GUIDE.md)
- [Filebeat 配置文件](conf/filebeat.yml)
- [go-stash 配置文件](../go-stash/etc/config.yaml)

## ✨ 配置特性

- ✅ 所有服务统一配置
- ✅ JSON 格式便于结构化分析
- ✅ 自动日志轮转和压缩
- ✅ 7 天自动清理
- ✅ 服务名称区分
- ✅ 完整的链路追踪支持
- ✅ 与 Filebeat/Kafka/ES 无缝集成
