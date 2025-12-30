# Go Zero Voice Agent Backend

基于 Go-Zero 框架构建的智能语音对话代理系统后端，支持文字对话、语音交互、RAG检索增强和实时通信等功能。

## 项目简介

Go Zero Voice Agent 是一个完整的微服务架构项目，提供：
- 🤖 大语言模型对话服务（支持多种LLM提供商）
- 🎤 语音识别与合成（ASR/TTS）
- 📚 RAG文档检索增强生成
- 💬 实时WebSocket通信
- 👤 用户认证与授权
- 📞 WebRTC语音通话

## 技术栈

- **框架**: Go-Zero v1.9.2
- **通信**: gRPC, WebSocket, RESTful API
- **数据库**: MySQL 8.0, PostgreSQL + pgvector
- **缓存**: Redis 6.2.5
- **消息队列**: Kafka, Asynq
- **对象存储**: MinIO
- **服务发现**: Etcd
- **日志系统**: ELK Stack (Elasticsearch + Logstash + Kibana)
- **监控**: Prometheus + Grafana
- **容器化**: Docker Compose

## 项目结构

```
go-zero-voice-agent-backend/
├── app/                          # 应用服务目录
│   ├── usercenter/              # 用户中心服务
│   │   ├── cmd/
│   │   │   ├── api/            # HTTP API服务 (端口: 3081)
│   │   │   └── rpc/            # gRPC服务 (端口: 4081)
│   │   └── model/              # 数据模型
│   ├── llm/                     # LLM对话服务
│   │   ├── cmd/
│   │   │   ├── api/            # HTTP API服务 (端口: 3082)
│   │   │   └── rpc/            # gRPC服务 (端口: 4082)
│   │   └── model/
│   ├── voicechat/              # 语音聊天服务
│   │   ├── cmd/
│   │   │   ├── api/            # HTTP API服务 (端口: 3083)
│   │   │   └── rpc/            # gRPC服务 (端口: 4083)
│   │   └── model/
│   ├── rag/                     # RAG检索服务
│   │   ├── cmd/
│   │   │   ├── api/            # HTTP API服务 (端口: 3084)
│   │   │   └── rpc/            # gRPC服务 (端口: 4084)
│   │   └── model/
│   ├── chatroom/               # 聊天室WebSocket服务
│   │   ├── cmd/api/
│   │   └── internal/websocket/
│   └── mqueue/                 # 消息队列服务
│       ├── cmd/job/            # 异步任务处理
│       └── cmd/scheduler/      # 定时任务调度
├── pkg/                         # 公共包
│   ├── middleware/             # 中间件（JWT认证等）
│   ├── result/                 # HTTP响应处理
│   ├── xerr/                   # 错误码定义
│   ├── tool/                   # 工具函数
│   └── uniqueid/               # 分布式ID生成
├── deploy/                      # 部署配置
│   ├── docker-compose/
│   ├── nginx/
│   └── sql/                    # 数据库初始化脚本
├── data/                        # 运行时数据
│   └── server/                 # 编译后的二进制文件
└── modd.conf                   # 热重载配置

```

## 微服务架构

### 1. User Center (用户中心)
**路由前缀**: `/usercenter/v1`
**端口**: API 3081 | RPC 4081

功能：
- 用户注册与登录（邮箱、微信小程序）
- JWT Token认证
- 用户信息管理
- 邮箱验证码发送

主要接口：
```
POST   /user/login          # 用户登录
POST   /user/register       # 用户注册
POST   /user/sendCode       # 发送验证码
GET    /user/auth           # Token验证
GET    /user/info           # 获取用户信息
POST   /user/wxMiniAuth     # 微信小程序认证
```

### 2. LLM Service (大语言模型服务)
**路由前缀**: `/llm/v1`
**端口**: API 3082 | RPC 4082

功能：
- 文字对话（支持流式响应）
- 聊天会话管理
- LLM配置管理
- 支持多种模型提供商（Aliyun DashScope、OpenAI等）
- Tool Call工具调用

主要接口：
```
POST   /chat/text                 # 文字对话
POST   /chat-message/list         # 查询聊天消息
GET    /chat-session/:id          # 获取会话详情
POST   /chat-session/list         # 查询会话列表
POST   /config/create             # 创建LLM配置
POST   /config/list               # 查询配置列表
```

### 3. Voice Chat (语音聊天)
**路由前缀**: `/voice/v1`
**端口**: API 3083 | RPC 4083

功能：
- ASR配置管理（语音识别）
- TTS配置管理（文字转语音）
- WebRTC通话支持
- Rust PBX集成

主要接口：
```
POST   /asr/config              # 创建ASR配置
GET    /asr/config/:id          # 获取ASR配置
POST   /asr/config/list         # 查询ASR配置列表
POST   /tts/config              # 创建TTS配置
GET    /tts/config/:id          # 获取TTS配置
POST   /tts/config/list         # 查询TTS配置列表
GET    /voice/chat/start        # 启动语音聊天
```

### 4. Chatroom (聊天室)
**路由前缀**: `/ws/v1`
**端口**: 3083 (与Voice Chat共用)

功能：
- WebSocket连接管理
- 多用户实时通信
- 连接池与分片管理
- 消息队列支持
- 集群模式支持

主要接口：
```
GET    /ws/open                 # 创建WebSocket连接
```

WebSocket配置项：
- 最大连接数、心跳间隔
- 消息缓冲大小
- 压缩支持、集群模式
- 分片管理和广播优化

### 5. RAG Service (检索增强生成)
**路由前缀**: `/rag/v1`
**端口**: API 3084 | RPC 4084

功能：
- 文档上传与向量化
- 文档分片管理
- 向量检索
- Python FastAPI后端集成

主要接口：
```
POST   /doc/upload              # 上传文件并向量化
POST   /doc/list                # 查询文档列表
GET    /doc/:id                 # 获取文档详情
DELETE /doc/:id                 # 删除文档
POST   /doc/:fileId/chunks      # 查询文件切片
```

### 6. Message Queue (消息队列)
功能：
- 异步任务处理
- 定时任务调度
- Redis/Kafka集成

## 快速开始

### 环境要求

- Go 1.21+
- Docker & Docker Compose
- MySQL 8.0
- PostgreSQL 16 (with pgvector)
- Redis 6.2+

### 安装步骤

1. **克隆项目**
```bash
git clone <repository-url>
cd go-zero-voice-agent-backend
```

2. **配置环境变量**
```bash
cp .env.example .env
# 编辑 .env 文件，配置必要的环境变量
```

3. **启动基础设施服务**
```bash
docker-compose -f docker-compose-env.yml up -d
```

这将启动以下服务：
- PostgreSQL (端口: 5432)
- MySQL (端口: 3306)
- Redis (端口: 36379)
- Etcd (端口: 2379)
- MinIO (端口: 9000/9001)
- Elasticsearch (端口: 9200)
- Kibana (端口: 5601)
- Kafka (端口: 9092)

4. **初始化数据库**
```bash
# 导入SQL初始化脚本
mysql -h 127.0.0.1 -P 3306 -u root -p < deploy/sql/gzva_usercenter.sql
mysql -h 127.0.0.1 -P 3306 -u root -p < deploy/sql/gzva_llmservice.sql
mysql -h 127.0.0.1 -P 3306 -u root -p < deploy/sql/gzva_voicechat.sql
mysql -h 127.0.0.1 -P 3306 -u root -p < deploy/sql/gzva_rag.sql
```

5. **安装依赖**
```bash
go mod download
```

6. **编译并运行服务**

方式一：使用热重载（开发环境推荐）
```bash
# 需要先安装 modd
go install github.com/cortesi/modd/cmd/modd@latest

# 启动所有服务（热重载）
modd
```

方式二：手动编译运行
```bash
# 编译用户中心API服务
go build -o data/server/usercenter-api app/usercenter/cmd/api/usercenter.go
./data/server/usercenter-api -f app/usercenter/cmd/api/etc/usercenter.yaml

# 编译用户中心RPC服务
go build -o data/server/usercenter-rpc app/usercenter/cmd/rpc/usercenter.go
./data/server/usercenter-rpc -f app/usercenter/cmd/rpc/etc/usercenter.yaml

# 其他服务类似...
```

7. **验证服务**
```bash
# 测试用户注册接口
curl -X POST http://localhost:3081/usercenter/v1/user/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"123456","code":"123456"}'
```

## 环境变量配置

主要环境变量（详见 `.env.example`）：

### 基础服务
```bash
# Etcd配置
ETCD_HOSTS=127.0.0.1:2379

# Redis配置
REDIS_HOST=127.0.0.1:36379
REDIS_PASSWORD=

# MinIO配置
MINIO_ENDPOINT=127.0.0.1:9000
MINIO_ACCESS_KEY_ID=minioadmin
MINIO_SECRET_ACCESS_KEY=minioadmin
```

### 用户中心
```bash
USERCENTER_API_HOST=0.0.0.0
USERCENTER_API_PORT=3081
USERCENTER_RPC_HOST=0.0.0.0
USERCENTER_RPC_PORT=4081
USERCENTER_DB_DSN=root:password@tcp(127.0.0.1:3306)/gzva_usercenter?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai
```

### JWT配置
```bash
JWT_ACCESS_SECRET=your-secret-key
JWT_ACCESS_EXPIRE=604800  # 7天
```

### 邮件配置
```bash
EMAIL_HOST=smtp.163.com
EMAIL_PORT=465
EMAIL_USERNAME=your-email@163.com
EMAIL_PASSWORD=your-smtp-password
EMAIL_FROM=your-email@163.com
```

### LLM配置
```bash
ALIYUN_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
ALIYUN_API_KEY=your-api-key
```

### WebSocket配置
```bash
WS_MAX_CONNECTIONS=10000
WS_HEARTBEAT_INTERVAL=30
WS_WRITE_WAIT=10
WS_PONG_WAIT=60
WS_READ_BUFFER_SIZE=1024
WS_WRITE_BUFFER_SIZE=1024
```

## API响应规范

### 成功响应
```json
{
  "code": 200,
  "msg": "OK",
  "data": {
    // 响应数据
  }
}
```

### 错误响应
```json
{
  "code": 100001,
  "msg": "错误信息"
}
```

### 错误码规范
错误码采用6位数字，前3位代表业务模块，后3位代表具体功能：

- `100xxx` - 全局错误
  - `100001` - 服务器内部错误
  - `100002` - 参数错误
  - `100003` - 数据库错误

- `200xxx` - 用户中心错误
  - `200001` - 用户名已存在
  - `200002` - 密码错误
  - `200003` - 验证码错误

- `300xxx` - LLM模块错误

## 认证与授权

### JWT认证流程

1. 用户登录获取Token
```bash
curl -X POST http://localhost:3081/usercenter/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'
```

响应：
```json
{
  "code": 200,
  "msg": "OK",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "accessExpire": 1704067200,
    "refreshAfter": 1703980800
  }
}
```

2. 携带Token访问受保护接口
```bash
curl -X GET http://localhost:3081/usercenter/v1/user/info \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 中间件配置

在 `api` 文件中配置 JWT 认证：
```go
@server(
    jwt: Auth
    middleware: CommonJwtAuthMiddleware
)
```

## 数据库设计

### 主要数据库

- **gzva_usercenter** - 用户中心数据库
  - `user` - 用户表
  - `user_auth` - 用户认证信息表

- **gzva_llmservice** - LLM服务数据库
  - `chat_config` - LLM配置表
  - `chat_session` - 聊天会话表
  - `chat_message` - 聊天消息表

- **gzva_voicechat** - 语音聊天数据库
  - `asr_config` - ASR配置表
  - `tts_config` - TTS配置表

- **gzva_rag** - RAG服务数据库
  - `document` - 文档表
  - `document_chunk` - 文档分片表（存储在PostgreSQL）

## 开发指南

### 代码生成

使用 goctl 生成代码：

```bash
# 生成API代码
goctl api go -api app/usercenter/cmd/api/desc/usercenter.api -dir app/usercenter/cmd/api

# 生成RPC代码
goctl rpc protoc app/usercenter/cmd/rpc/pb/usercenter.proto --go_out=. --go-grpc_out=. --zrpc_out=.

# 生成Model代码
goctl model mysql datasource -url="root:password@tcp(127.0.0.1:3306)/gzva_usercenter" -table="user" -dir="app/usercenter/model"
```

### 项目规范

1. **目录结构**：遵循 go-zero 官方推荐结构
2. **命名规范**：使用驼峰命名，包名小写
3. **错误处理**：统一使用 `xerr` 包定义错误码
4. **日志记录**：使用 go-zero 内置的 logx
5. **配置管理**：使用 YAML 配置文件 + 环境变量

### 测试

```bash
# 运行单元测试
go test ./...

# 运行特定模块测试
go test ./app/usercenter/...

# 测试覆盖率
go test -cover ./...
```

## 部署

### Docker部署

```bash
# 构建镜像
docker build -t gzva-usercenter-api:latest -f deploy/docker/Dockerfile.usercenter-api .

# 运行容器
docker run -d \
  --name gzva-usercenter-api \
  -p 3081:3081 \
  -v ./app/usercenter/cmd/api/etc:/app/etc \
  gzva-usercenter-api:latest
```

### Docker Compose部署

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f usercenter-api

# 停止服务
docker-compose down
```

## 监控与日志

### 日志查询

访问 Kibana: http://localhost:5601

1. 创建索引模式：`go-stash-*`
2. 查询日志：使用 KQL 语法过滤

### 指标监控

访问 Grafana: http://localhost:3000

- 默认用户名/密码：admin/admin
- 导入 go-zero 监控面板

### 链路追踪

配置 Jaeger/Zipkin 端点：
```yaml
Telemetry:
  Name: usercenter-api
  Endpoint: http://localhost:14268/api/traces
  Sampler: 1.0
  Batcher: jaeger
```

## 常见问题

### 1. 服务无法启动
- 检查端口是否被占用
- 检查配置文件路径是否正确
- 检查数据库连接是否正常

### 2. JWT认证失败
- 检查 `JWT_ACCESS_SECRET` 是否配置
- 检查Token是否过期
- 检查请求头格式：`Authorization: Bearer <token>`

### 3. 数据库连接错误
- 检查DSN配置是否正确
- 检查数据库服务是否启动
- 检查网络连接是否正常

### 4. WebSocket连接失败
- 检查Nginx配置是否支持WebSocket升级
- 检查防火墙设置
- 检查连接数是否超过限制

## 性能优化

### 数据库优化
- 使用连接池
- 添加适当索引
- 使用缓存减少数据库查询

### 缓存策略
- 用户信息缓存（TTL: 1小时）
- 配置信息缓存（TTL: 24小时）
- 使用Redis Pipeline批量操作

### 并发优化
- 使用协程池处理并发请求
- 使用消息队列处理异步任务
- 使用分布式锁防止并发冲突

## 贡献指南

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 LICENSE 文件

## 联系方式

- 项目主页: [GitHub Repository]
- 问题反馈: [GitHub Issues]
- 邮箱: [your-email@example.com]

## 致谢

- [go-zero](https://github.com/zeromicro/go-zero) - 微服务框架
- [Aliyun DashScope](https://dashscope.aliyun.com/) - 大模型服务
- 所有贡献者

---

**注意**: 本项目仅供学习和研究使用，生产环境部署前请做好安全加固和性能测试。
