# Filebeat 日志收集配置说明

## 架构说明

本项目使用以下架构收集和存储日志：

```
Go 应用 (宿主机) -> 日志文件 -> Filebeat (容器) -> Kafka (容器) -> go-stash (容器) -> Elasticsearch (容器) -> Kibana (容器)
```

## 前置条件

1. Go 应用在宿主机运行（非容器化）
2. 中间件（Kafka、Elasticsearch、Kibana 等）运行在 Docker 容器中
3. 日志文件需要输出到统一目录

## 配置步骤

### 1. 确保日志目录存在

```bash
cd /home/zhang/go-zero-voice-agent/go-zero-voice-agent-backend
mkdir -p logs
```

### 2. 配置 Go 应用的日志输出

在各个服务的配置文件中添加 Log 配置，例如：

```yaml
Log:
  ServiceName: usercenter-api
  Mode: file
  Path: logs
  Level: info
  Encoding: json
  KeepDays: 7
```

### 3. 启动中间件服务

```bash
# 启动日志收集栈
docker compose -f docker-compose-env.yml up -d kafka elasticsearch kibana

# 等待服务启动（约 30-60 秒）
docker compose -f docker-compose-env.yml ps
```

### 4. 创建 Kafka Topic

```bash
# 进入 Kafka 容器
docker exec -it kafka bash

# 创建 topic
/opt/kafka/bin/kafka-topics.sh --create \
  --bootstrap-server localhost:9092 \
  --topic looklook-log \
  --partitions 3 \
  --replication-factor 1

# 查看创建的 topic
/opt/kafka/bin/kafka-topics.sh --list --bootstrap-server localhost:9092

# 退出容器
exit
```

### 5. 启动 Filebeat 和 go-stash

```bash
docker compose -f docker-compose-env.yml up -d filebeat go-stash
```

### 6. 验证日志收集

```bash
# 查看 Filebeat 日志
docker logs -f filebeat

# 查看 go-stash 日志
docker logs -f go-stash

# 查看 Kafka 消息
docker exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic looklook-log \
  --from-beginning \
  --max-messages 10
```

### 7. 在 Kibana 中查看日志

1. 访问 Kibana: http://localhost:5601
2. 进入 "Management" -> "Stack Management" -> "Index Patterns"
3. 创建索引模式: `go-zero-voice-agent-*`
4. 选择时间字段: `@timestamp`
5. 在 "Discover" 中查看日志

## 日志格式说明

Go-Zero 默认输出 JSON 格式日志，包含以下字段：

- `@timestamp`: 时间戳
- `level`: 日志级别 (info, error, debug, etc.)
- `content`: 日志内容
- `caller`: 调用位置
- `span`: 链路追踪 ID
- `trace`: 追踪 ID

## 故障排查

### Filebeat 无法启动

```bash
# 检查配置文件语法
docker run --rm -v $(pwd)/deploy/filebeat/conf/filebeat.yml:/usr/share/filebeat/filebeat.yml \
  elastic/filebeat:8.12.2 test config -e

# 查看详细日志
docker logs filebeat
```

### 日志未被收集

1. 检查日志文件是否存在：`ls -la logs/`
2. 检查文件权限：Filebeat 容器以 root 用户运行，应该有读取权限
3. 检查 Filebeat 日志：`docker logs filebeat`

### Kafka 连接失败

```bash
# 检查 Kafka 是否运行
docker compose -f docker-compose-env.yml ps kafka

# 检查网络连接
docker exec filebeat ping kafka
```

### Elasticsearch 无数据

1. 检查 go-stash 日志：`docker logs go-stash`
2. 检查 Kafka 中是否有消息
3. 检查 Elasticsearch 索引：

```bash
curl -X GET "localhost:9200/_cat/indices?v"
curl -X GET "localhost:9200/go-zero-voice-agent-*/_search?pretty"
```

## 性能优化建议

1. **Filebeat**: 调整 `scan_frequency` 控制扫描频率
2. **Kafka**: 增加分区数以提高并发处理能力
3. **Elasticsearch**: 设置适当的副本和分片策略
4. **日志轮转**: 配置 go-zero 的 `KeepDays` 自动清理旧日志

## 维护建议

1. 定期清理 Elasticsearch 旧索引
2. 监控 Kafka 磁盘使用
3. 配置日志告警规则
4. 设置日志保留策略
