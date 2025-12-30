#!/bin/bash

# Filebeat 日志收集系统启动脚本（改进版）

set -e

PROJECT_ROOT="/home/zhang/go-zero-voice-agent/go-zero-voice-agent-backend"
cd "$PROJECT_ROOT"

echo "=================================="
echo "启动 Filebeat 日志收集系统"
echo "=================================="

# 1. 创建日志目录
echo ""
echo "[1/8] 创建日志目录..."
mkdir -p logs
chmod 755 logs
echo "✓ 日志目录已创建: $(pwd)/logs"

# 2. 修复数据目录权限
echo ""
echo "[2/8] 修复数据目录权限..."
mkdir -p data/kafka data/es
chown -R 1000:1000 data/kafka data/es 2>/dev/null || true
echo "✓ Kafka 和 Elasticsearch 数据目录权限已修复"

# 3. 启动 Kafka
echo ""
echo "[3/8] 启动 Kafka..."
docker compose -f docker-compose-env.yml up -d kafka

echo "等待 Kafka 启动 (40秒)..."
sleep 40

# 检查 Kafka 状态
if docker logs kafka --tail 5 | grep -q "Kafka Server started"; then
    echo "✓ Kafka 启动成功"
else
    echo "⚠ Kafka 可能还在启动中，继续执行..."
fi

# 4. 创建 Kafka Topic
echo ""
echo "[4/8] 创建 Kafka Topic..."
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --create \
  --bootstrap-server kafka:9092 \
  --topic looklook-log \
  --partitions 3 \
  --replication-factor 1 \
  --if-not-exists 2>/dev/null || echo "Topic 可能已存在或 Kafka 还在初始化"

# 验证 Topic
echo ""
echo "验证 Topic 创建..."
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
  --list \
  --bootstrap-server kafka:9092 2>/dev/null || echo "暂时无法列出 Topic"

# 5. 启动 Elasticsearch
echo ""
echo "[5/8] 启动 Elasticsearch..."
docker compose -f docker-compose-env.yml up -d elasticsearch

echo "等待 Elasticsearch 启动 (40秒)..."
sleep 40

# 检查 ES 状态
echo "检查 Elasticsearch 状态..."
curl -s http://localhost:9200/_cluster/health?pretty 2>/dev/null || echo "Elasticsearch 还在启动中..."

# 6. 启动 Kibana
echo ""
echo "[6/8] 启动 Kibana..."
docker compose -f docker-compose-env.yml up -d kibana

# 7. 启动 Filebeat
echo ""
echo "[7/8] 启动 Filebeat..."
docker compose -f docker-compose-env.yml up -d filebeat

sleep 5

# 检查 Filebeat 状态
if docker ps | grep -q filebeat; then
    echo "✓ Filebeat 启动成功"
    echo ""
    echo "Filebeat 最近日志:"
    docker logs filebeat --tail 10
else
    echo "✗ Filebeat 启动失败"
    docker logs filebeat --tail 20
fi

# 8. 启动 go-stash
echo ""
echo "[8/8] 启动 go-stash..."
docker compose -f docker-compose-env.yml up -d go-stash

echo ""
echo "=================================="
echo "✓ 服务启动完成！"
echo "=================================="
echo ""
echo "服务状态:"
docker compose -f docker-compose-env.yml ps kafka elasticsearch kibana filebeat go-stash

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "访问地址:"
echo "  🌐 Kibana:        http://localhost:5601"
echo "  🔍 Elasticsearch: http://localhost:9200"
echo ""
echo "有用的命令:"
echo "  📝 查看 Filebeat 日志:    docker logs -f filebeat"
echo "  📝 查看 go-stash 日志:    docker logs -f go-stash"
echo "  📝 查看 Kafka 消息:"
echo "     docker exec kafka /opt/kafka/bin/kafka-console-consumer.sh \\"
echo "       --bootstrap-server kafka:9092 \\"
echo "       --topic looklook-log \\"
echo "       --max-messages 10"
echo ""
echo "  📊 检查 ES 索引:"
echo "     curl http://localhost:9200/_cat/indices?v"
echo ""
echo "  🔄 查看所有服务:"
echo "     docker compose -f docker-compose-env.yml ps"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
