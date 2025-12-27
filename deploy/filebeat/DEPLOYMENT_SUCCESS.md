# ✅ Filebeat 日志收集系统部署成功！

## 🎉 部署状态

所有日志收集服务已成功启动并运行：

```
✓ Kafka           - 消息队列 (端口: 9092)
✓ Elasticsearch   - 日志存储 (端口: 9200, 9300)
✓ Kibana          - 可视化界面 (端口: 5601)
✓ Filebeat        - 日志收集器
✓ go-stash        - Kafka 消费者 → ES 写入
```

## 📊 系统架构

```
Go 应用 (宿主机)
    ↓ 写入日志文件 (JSON 格式)
logs/*.log
    ↓ Filebeat 收集
Kafka (topic: looklook-log)
    ↓ go-stash 消费
Elasticsearch
    ↓ Kibana 查询展示
用户查看日志
```

## ✅ 已完成的配置

### 1. 服务配置 (10个服务)

**API 服务 (5个):**
- ✅ chatroom-api
- ✅ usercenter-api
- ✅ voicechat-api
- ✅ llm-api
- ✅ rag-api

**RPC 服务 (4个):**
- ✅ usercenter-rpc
- ✅ voicechat-rpc
- ✅ llm-rpc
- ✅ rag-rpc

**Job 服务 (1个):**
- ✅ mqueue-job

所有服务统一配置：
```yaml
Log:
  ServiceName: <服务名>
  Mode: file
  Path: logs
  Level: info
  Encoding: json
  KeepDays: 7
  Compress: true
```

### 2. Filebeat 配置

**位置:** `deploy/filebeat/conf/filebeat.yml`

**配置要点:**
- 收集路径: `logs/*.log` 和 `logs/*/*.log`
- JSON 格式解析
- 输出到 Kafka (kafka:9092)
- Topic: `looklook-log`

### 3. go-stash 配置

**位置:** `deploy/go-stash/etc/config.yaml`

**功能:**
- 从 Kafka 消费日志
- 写入 Elasticsearch
- 索引名称: `go-zero-voice-agent-{yyyy-MM-dd}`

### 4. Kafka Topic

**名称:** `looklook-log`
**分区:** 3
**副本:** 1

## 🚀 快速开始

### 方式一：一键启动（推荐）

```bash
cd /home/zhang/go-zero-voice-agent/go-zero-voice-agent-backend
./deploy/filebeat/start-logging.sh
```

### 方式二：重启服务

```bash
# 重启日志收集栈
docker compose -f docker-compose-env.yml restart kafka elasticsearch kibana filebeat go-stash

# 查看状态
docker compose -f docker-compose-env.yml ps
```

## 📝 使用日志系统

### 1. 启动你的 Go 应用

启动任何配置了日志的服务，日志将自动写入 `logs/` 目录：

```bash
cd app/usercenter/cmd/api
go run usercenter.go -f etc/usercenter.yaml
```

### 2. 查看日志文件

```bash
# 查看生成的日志文件
ls -lh logs/

# 实时查看日志
tail -f logs/usercenter-api-access.log

# 查看 JSON 格式（需要安装 jq）
cat logs/usercenter-api-access.log | jq .
```

### 3. 在 Kibana 查看日志

1. **访问 Kibana:** http://localhost:5601

2. **创建索引模式:**
   - 进入 **Management** → **Stack Management** → **Index Patterns**
   - 点击 **Create index pattern**
   - 输入模式: `go-zero-voice-agent-*`
   - 选择时间字段: `@timestamp`
   - 点击 **Create**

3. **查看日志:**
   - 进入 **Discover**
   - 即可看到所有收集的日志

### 4. Kibana 搜索示例

```
# 查看错误日志
level: "error"

# 查看特定服务
log_source: "go-zero-voice-agent" AND content: *usercenter*

# 查看 HTTP 错误
status >= 400

# 查看慢请求
duration > 1000

# 查看特定 API
path: "/api/user/login"
```

## 🔍 验证日志收集

### 检查 Filebeat

```bash
# 查看 Filebeat 日志
docker logs -f filebeat

# 验证配置路径
docker logs filebeat 2>&1 | grep "Configured paths"
```

### 检查 Kafka

```bash
# 查看 Topic 列表
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list --bootstrap-server kafka:9092

# 查看消息（实时）
docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server kafka:9092 \
  --topic looklook-log \
  --max-messages 10
```

### 检查 Elasticsearch

```bash
# 查看索引列表
curl http://localhost:9200/_cat/indices?v

# 查看日志数据
curl http://localhost:9200/go-zero-voice-agent-*/_search?pretty

# 查看集群健康
curl http://localhost:9200/_cluster/health?pretty
```

### 检查 go-stash

```bash
# 查看 go-stash 日志
docker logs -f go-stash
```

## 🛠 故障排查

### Filebeat 未收集日志

**检查清单:**
1. ✅ 日志文件是否存在: `ls logs/`
2. ✅ 日志格式是否为 JSON
3. ✅ Filebeat 容器是否运行: `docker ps | grep filebeat`
4. ✅ 查看 Filebeat 日志: `docker logs filebeat`

### Kafka 无消息

**检查:**
```bash
# Filebeat 是否连接成功
docker logs filebeat | grep kafka

# Kafka 容器是否正常
docker logs kafka --tail 50
```

### Elasticsearch 无数据

**检查:**
```bash
# go-stash 是否正常消费
docker logs go-stash

# 检查索引
curl http://localhost:9200/_cat/indices?v
```

## 📚 相关文档

- [配置完成汇总](CONFIGURATION_SUMMARY.md)
- [日志配置详细说明](LOG_CONFIG_GUIDE.md)
- [Filebeat 配置](conf/filebeat.yml)
- [go-stash 配置](../go-stash/etc/config.yaml)

## 💡 提示

1. **日志格式必须是 JSON** - Go-Zero 默认支持
2. **日志路径必须是 `logs`** - 所有服务统一配置
3. **Filebeat 会自动跟踪文件位置** - 重启不会丢失数据
4. **Elasticsearch 自动创建索引** - 按日期分割
5. **旧日志自动压缩和清理** - KeepDays: 7

## ⚙️ 维护命令

```bash
# 查看所有服务状态
docker compose -f docker-compose-env.yml ps

# 重启某个服务
docker compose -f docker-compose-env.yml restart filebeat

# 停止日志收集系统
docker compose -f docker-compose-env.yml down kafka elasticsearch kibana filebeat go-stash

# 清理旧日志（谨慎使用）
find logs/ -name "*.log.*" -mtime +7 -delete

# 查看磁盘使用
du -sh data/es data/kafka logs/
```

---

## 🎯 下一步

现在你可以:

1. ✅ 启动你的 Go 应用服务
2. ✅ 在 Kibana 中实时查看和搜索日志
3. ✅ 设置日志告警规则
4. ✅ 创建日志分析仪表板

**日志收集系统已就绪！** 🚀
